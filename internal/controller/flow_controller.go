/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/controller/appbuilder"
	"github.com/jettisonproj/jettison-controller/internal/controller/sensorbuilder"
)

const (
	conditionType = "Ready"
)

// FlowReconciler reconciles a Flow object
type FlowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows/finalizers,verbs=update
// +kubebuilder:rbac:groups=argoproj.io,resources=sensors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Flow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *FlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	reconcilerlog := log.FromContext(ctx)
	flow := &v1alpha1.Flow{}
	if err := r.Get(ctx, req.NamespacedName, flow); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			reconcilerlog.Info("Flow resource not found. Ignoring deleted object")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reconcilerlog.Error(err, "unable to fetch Flow")
		return ctrl.Result{}, err
	}

	// Preprocess the Flow to parse the triggers and steps
	flowTriggers, flowSteps, err := flow.PreProcessFlow()
	if err != nil {
		reconcilerlog.Error(err, "error preprocessing Flow", "flow", flow)
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "PreProcessFailed",
			Message: fmt.Sprintf("Failed to preprocess Flow: %s", err),
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status after preprocess")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Reconcile the argocd AppProject and Application(s)
	detectedDrift := false
	projects, applications, err := appbuilder.BuildArgoApps(flowSteps)
	if err != nil {
		reconcilerlog.Error(err, "error building Flow applications", "flow", flow)
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "AppBuildFailed",
			Message: fmt.Sprintf("Failed to build Flow applications: %s", err),
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status after application build")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	for _, project := range projects {
		existingProject := &cdv1.AppProject{}
		existingNamespaceName := client.ObjectKey{
			Namespace: project.ObjectMeta.Namespace,
			Name:      project.ObjectMeta.Name,
		}

		reconcilerlog.Info("look up project", "nsname", existingNamespaceName)
		err = r.Get(ctx, existingNamespaceName, existingProject)
		if err != nil && !apierrors.IsNotFound(err) {
			reconcilerlog.Error(err, "failed to check for existing app project")
			// Let's return the error for the reconciliation be re-trigged again
			return ctrl.Result{}, err
		}

		existingProjectNotFound := err != nil

		// Note: The ownerRef for the AppProject cannot be set, since
		// these are in a separate namespace

		if existingProjectNotFound {
			if err := r.Create(ctx, project); err != nil {
				reconcilerlog.Error(err, "failed to create AppProject for Flow")
				return ctrl.Result{}, err
			}
			detectedDrift = true
		} else if !equality.Semantic.DeepDerivative(project.Spec, existingProject.Spec) {
			//
			// For discussion on the comparison here, see:
			// https://github.com/kubernetes-sigs/kubebuilder/issues/592
			//
			// some field set by the operator has changed
			existingProject.Spec = project.Spec
			if err := r.Update(ctx, existingProject); err != nil {
				reconcilerlog.Error(err, "failed to update AppProject for Flow")
				return ctrl.Result{}, err
			}
			detectedDrift = true
		} else {
			reconcilerlog.Info("no AppProject drift detected for Flow")
		}
	}

	for _, application := range applications {
		existingApplication := &cdv1.Application{}
		existingNamespaceName := client.ObjectKey{
			Namespace: application.ObjectMeta.Namespace,
			Name:      application.ObjectMeta.Name,
		}

		err = r.Get(ctx, existingNamespaceName, existingApplication)
		if err != nil && !apierrors.IsNotFound(err) {
			reconcilerlog.Error(err, "failed to check for existing Application")
			// Let's return the error for the reconciliation be re-trigged again
			return ctrl.Result{}, err
		}

		existingApplicationNotFound := err != nil

		// Set the ownerRef for the Application
		// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
		if err := ctrl.SetControllerReference(flow, application, r.Scheme); err != nil {
			reconcilerlog.Error(err, "unable to set Application owner reference")
			return ctrl.Result{}, err
		}

		if existingApplicationNotFound {
			if err := r.Create(ctx, application); err != nil {
				reconcilerlog.Error(err, "failed to create Application for Flow")
				return ctrl.Result{}, err
			}
			detectedDrift = true
		} else if !equality.Semantic.DeepDerivative(application.Spec, existingApplication.Spec) {
			//
			// For discussion on the comparison here, see:
			// https://github.com/kubernetes-sigs/kubebuilder/issues/592
			//
			// some field set by the operator has changed
			existingApplication.Spec = application.Spec
			if err := r.Update(ctx, existingApplication); err != nil {
				reconcilerlog.Error(err, "failed to update Application for Flow")
				return ctrl.Result{}, err
			}
			detectedDrift = true
		} else {
			reconcilerlog.Info("no Application drift detected for Flow")
		}
	}

	// Reconcile the Sensor
	existingSensor := &eventsv1.Sensor{}
	err = r.Get(ctx, req.NamespacedName, existingSensor)
	if err != nil && !apierrors.IsNotFound(err) {
		reconcilerlog.Error(err, "failed to check for existing Sensor")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	existingSensorNotFound := err != nil

	sensor, err := sensorbuilder.BuildSensor(flow, flowTriggers, flowSteps)
	if err != nil {
		reconcilerlog.Error(err, "failed to build Sensor", "flow", flow)
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "SensorBuildFailed",
			Message: fmt.Sprintf("Failed to build Sensor: %s", err),
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status after sensor build")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Set the ownerRef for the Sensor
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(flow, sensor, r.Scheme); err != nil {
		reconcilerlog.Error(err, "unable to set Sensor owner reference")
		return ctrl.Result{}, err
	}

	if existingSensorNotFound {
		if err := r.Create(ctx, sensor); err != nil {
			reconcilerlog.Error(err, "failed to create Sensor for Flow")
			return ctrl.Result{}, err
		}
		detectedDrift = true
	} else if !equality.Semantic.DeepDerivative(sensor.Spec, existingSensor.Spec) {
		//
		// For discussion on the comparison here, see:
		// https://github.com/kubernetes-sigs/kubebuilder/issues/592
		//
		// some field set by the operator has changed
		existingSensor.Spec = sensor.Spec
		if err := r.Update(ctx, existingSensor); err != nil {
			reconcilerlog.Error(err, "failed to update Sensor for Flow")
			return ctrl.Result{}, err
		}
		detectedDrift = true
	} else {
		reconcilerlog.Info("no Sensor drift detected for Flow")
	}

	if detectedDrift {
		reconcilerlog.Info("successfully synced Flow")
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "FlowSynced",
			Message: "Successfully synced Flow resource",
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status after sync")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Flow{}).
		Owns(&eventsv1.Sensor{}).
		Named("flow").
		Complete(r)
}
