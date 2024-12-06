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

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/controller/sensor"
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

	existingSensor := &eventsv1.Sensor{}
	err := r.Get(ctx, req.NamespacedName, existingSensor)
	if err != nil && !apierrors.IsNotFound(err) {
		reconcilerlog.Error(err, "failed to check for existing Sensor")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	existingSensorNotFound := err != nil
	sensor, err := sensor.BuildSensor(flow)
	if err != nil {
		reconcilerlog.Error(err, "failed to build Sensor", "flow", flow)
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "SensorGenFailed",
			Message: fmt.Sprintf("Failed to build Sensor: %s", err),
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Set the ownerRef for the Deployment
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
		reconcilerlog.Info("created new Sensor for Flow")
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "SensorCreated",
			Message: "Successfully created sensor resource",
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// For discussion on the comparison here, see:
	// https://github.com/kubernetes-sigs/kubebuilder/issues/592
	if !equality.Semantic.DeepDerivative(sensor.Spec, existingSensor.Spec) {
		// some field set by the operator has changed
		existingSensor.Spec = sensor.Spec
		if err := r.Update(ctx, existingSensor); err != nil {
			reconcilerlog.Error(err, "failed to update Sensor for Flow")
			return ctrl.Result{}, err
		}
		reconcilerlog.Info("updated Sensor for Flow")
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "SensorUpdated",
			Message: "Successfully updated sensor resource",
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	reconcilerlog.Info("reconciler called for Flow, but no drift was detected")

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
