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
	eventsourcev1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
	"github.com/jettisonproj/jettison-controller/internal/controller/appbuilder"
	"github.com/jettisonproj/jettison-controller/internal/controller/eventsourcebuilder"
	"github.com/jettisonproj/jettison-controller/internal/controller/ghsettings"
	"github.com/jettisonproj/jettison-controller/internal/controller/sensorbuilder"
	"github.com/jettisonproj/jettison-controller/internal/gitutil"
)

const (
	conditionType = "Ready"
)

// FlowReconciler reconciles a Flow object
type FlowReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	GitHubTransport *ghinstallation.AppsTransport
	GitHubClient    *github.Client

	// Multiple Flows can share a namespace
	// Keep track of synced namespaces to avoid unnecessary updates
	SyncedNamespaces map[string]bool

	// Multiple Flows can share an AppProject
	// Keep track of synced AppProject names to avoid unnecessary updates
	SyncedProjects map[string]bool

	// Multiple Flows can share repos which are propagated to shared resources
	// e.g. EventSource and GitHub Settings
	// Keep track of synced repoOrg/repoName to avoid unnecessary updates
	SyncedRepos map[string]map[string]bool
}

// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workflows.jettisonproj.io,resources=flows/finalizers,verbs=update

// Jettison manages argo resources to manage deployments
// +kubebuilder:rbac:groups=argoproj.io,resources=applications;appprojects;clusterworkflowtemplates;eventsources;sensors,verbs=get;list;watch;create;update;patch;delete

// Jettison reads deployed resources to provide updated info to users
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=rollouts;workflows,verbs=get;list;watch

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

	detectedDrift := false

	if !r.SyncedNamespaces[flow.Namespace] {
		// todo: ideally the external deps could be moved to here, but currently
		// non-standard fields are used
		// Reconcile the EventBus ./apps/argo/argo-events/common/event-bus.yaml
		//   dynamic: namespace
		// Reconcile the Sensor RBAC https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/sensor-rbac.yaml
		//   dynamic: namespace
		//   description: Sensor needs RBAC to create workflows
		// Reconcile the Workflow RBAC https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/workflow-rbac.yaml
		//   dynamic: namespace
		//   description: the workflow manager pods themselves need to update the workflowtaskresults
		// Reconcile the ServiceAccount ./apps/argo/argo-events/common/serviceaccount-deploy-step-executor.yaml
		// Reconcile the Role ./apps/argo/argo-events/common/role-deploy-step-executor.yaml
		// Reconcile the RoleBinding ./apps/argo/argo-events/common/rolebinding-deploy-step-executor.yaml
		//   dynamic: namespace
		//   description: the actual deploy step pods may need permissions (e.g. ArgoCD deploy step monitors resources after deploy)
		r.SyncedNamespaces[flow.Namespace] = true
	}

	// Reconcile the argocd AppProject and Application(s)
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

		projectName := project.ObjectMeta.Name
		if r.SyncedProjects[projectName] {
			continue
		}

		existingProject := &cdv1.AppProject{}
		existingNamespaceName := client.ObjectKey{
			Namespace: project.ObjectMeta.Namespace,
			Name:      projectName,
		}

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

		r.SyncedProjects[projectName] = true
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
	existingSensor := &sensorsv1.Sensor{}
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

	// Reconcile the EventSource
	repoOrg, repoName, err := getTriggerRepoName(flow.Name, flowTriggers)
	if err != nil {
		reconcilerlog.Error(err, "failed to get repo name for EventSource", "flow", flow)
		condition := metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionFalse,
			Reason:  "EventSourceFailed",
			Message: fmt.Sprintf("Failed to get repo name for EventSource: %s", err),
		}
		if err := r.setCondition(ctx, req, flow, condition); err != nil {
			reconcilerlog.Error(err, "failed to update Flow status after event source repo name")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	syncedRepoNames, foundSyncedRepoOrg := r.SyncedRepos[repoOrg]
	if !foundSyncedRepoOrg || !syncedRepoNames[repoName] {

		eventSource := eventsourcebuilder.BuildEventSource(repoOrg, repoName)
		existingEventSource := &eventsourcev1.EventSource{}
		existingNamespaceName := client.ObjectKey{
			Namespace: eventSource.ObjectMeta.Namespace,
			Name:      eventSource.ObjectMeta.Name,
		}
		err = r.Get(ctx, existingNamespaceName, existingEventSource)
		if err != nil && !apierrors.IsNotFound(err) {
			reconcilerlog.Error(err, "failed to check for existing event source")
			// Let's return the error for the reconciliation be re-trigged again
			return ctrl.Result{}, err
		}

		existingEventSourceNotFound := err != nil

		// Note: The ownerRef for the EventSource cannot be set, since
		// these are in a separate namespace

		if existingEventSourceNotFound {
			if err := r.Create(ctx, eventSource); err != nil {
				reconcilerlog.Error(err, "failed to create EventSource for Flow")
				return ctrl.Result{}, err
			}
			detectedDrift = true
		} else {
			needsUpdate := mergeOwnedRepos(existingEventSource, repoOrg, repoName)
			if needsUpdate {
				if err := r.Update(ctx, existingEventSource); err != nil {
					reconcilerlog.Error(err, "failed to update EventSource for Flow")
					return ctrl.Result{}, err
				}
				detectedDrift = true
			} else {
				reconcilerlog.Info("no EventSource drift detected for Flow")
			}
		}

		// Reconcile GitHub Settings
		err := ghsettings.SyncGitHubSettingsForRepo(
			ctx,
			r.GitHubTransport,
			r.GitHubClient,
			repoOrg,
			repoName,
		)
		if err != nil {
			reconcilerlog.Error(err, "failed to sync GitHub settings for Flow")
			return ctrl.Result{}, err
		}

		if !foundSyncedRepoOrg {
			syncedRepoNames = make(map[string]bool)
			r.SyncedRepos[repoOrg] = syncedRepoNames
		}
		syncedRepoNames[repoName] = true
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
	// todo could consider adding additional child objects (e.g. application)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Flow{}).
		Owns(&sensorsv1.Sensor{}).
		Named("flow").
		Complete(r)
}

func getTriggerRepoName(
	flowName string,
	flowTriggers []v1alpha1base.BaseTrigger,
) (string, string, error) {

	triggerRepoUrl, err := getTriggerRepoUrl(flowName, flowTriggers)
	if err != nil {
		return "", "", err
	}
	return gitutil.GetRepoOrgName(triggerRepoUrl)
}

func getTriggerRepoUrl(
	flowName string,
	flowTriggers []v1alpha1base.BaseTrigger,
) (string, error) {
	for _, flowTrigger := range flowTriggers {
		switch trigger := flowTrigger.(type) {
		case *v1alpha1.GitHubPullRequestTrigger:
			return trigger.RepoUrl, nil
		case *v1alpha1.GitHubPushTrigger:
			return trigger.RepoUrl, nil
		default:
			return "", fmt.Errorf("unknown trigger type for flow %s: %T", flowName, trigger)
		}
	}
	return "", fmt.Errorf("did not find trigger for flow: %s", flowName)
}

func mergeOwnedRepos(
	eventSource *eventsourcev1.EventSource,
	repoOrg string,
	repoName string,
) bool {
	githubEventSources := eventSource.Spec.Github
	githubEventSource := githubEventSources[eventsourcebuilder.EventSourceSectionName]
	ownedRepos := githubEventSource.Repositories
	for i := range ownedRepos {
		if ownedRepos[i].Owner == repoOrg {
			for j := range ownedRepos[i].Names {
				if ownedRepos[i].Names[j] == repoName {
					return false
				}
			}
			ownedRepos[i].Names = append(ownedRepos[i].Names, repoName)
			githubEventSource.Repositories = ownedRepos
			githubEventSources[eventsourcebuilder.EventSourceSectionName] = githubEventSource
			return true
		}
	}

	ownedRepos = append(ownedRepos, eventsourcev1.OwnedRepositories{
		Owner: repoOrg,
		Names: []string{repoName},
	})
	githubEventSource.Repositories = ownedRepos
	githubEventSources[eventsourcebuilder.EventSourceSectionName] = githubEventSource
	return true
}
