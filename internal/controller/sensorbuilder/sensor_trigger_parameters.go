package sensorbuilder

import (
	"fmt"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

var (
	// The global workflow parameters. These will be available for each DAGTask to consume.
	// Each trigger is responsible for providing these via TriggerParameters
	globalWorkflowParameters = []workflowsv1.Parameter{
		// Parameter 0
		{
			Name: "repo",
		},
		// Parameter 1
		{
			Name: "revision",
		},
		// Parameter 2
		{
			Name: "repo-short",
		},
		// Parameter 3
		{
			Name: "revision-ref",
		},
		// Parameter 4
		{
			Name: "revision-title",
		},
		// Parameter 5
		{
			Name: "revision-author",
		},
	}
	// The PR workflow parameters. These will be available for each PR DAGTask to consume.
	// Each PR trigger is responsible for providing these via TriggerParameters
	prWorkflowParameters = []workflowsv1.Parameter{
		// Parameter 6
		{
			Name: "base-revision",
		},
		// Parameter 7
		{
			Name: "base-revision-ref",
		},
		// Parameter 8
		{
			Name: "revision-number",
		},
	}
)

func getTriggerParameters(flowTriggers []v1alpha1base.BaseTrigger) (
	[]workflowsv1.Parameter,
	[]eventsv1.TriggerParameter,
	error,
) {

	workflowParameters := make(
		[]workflowsv1.Parameter,
		len(globalWorkflowParameters),
		len(globalWorkflowParameters)+len(prWorkflowParameters),
	)
	copy(workflowParameters, globalWorkflowParameters)

	var triggerParameters []eventsv1.TriggerParameter
	for i := range flowTriggers {
		switch trigger := flowTriggers[i].(type) {
		case *v1alpha1.GitHubPullRequestTrigger:
			triggerParameters = append(
				triggerParameters,
				// Parameter 0: Set the repo url in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.repo.clone_url",
					},
					Dest: "spec.arguments.parameters.0.value",
				},
				// Parameter 1: Set the revision to check out
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.sha",
					},
					Dest: "spec.arguments.parameters.1.value",
				},
				// Parameter 2: Set the repo full_name in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.repo.full_name",
					},
					Dest: "spec.arguments.parameters.2.value",
				},
				// Parameter 3: Set the revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.ref",
					},
					Dest: "spec.arguments.parameters.3.value",
				},
				// Parameter 4: Set the revision title
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.title",
					},
					Dest: "spec.arguments.parameters.4.value",
				},
				// Parameter 5: Set the revision author
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.user.login",
					},
					Dest: "spec.arguments.parameters.5.value",
				},
				// Parameter 6: Set the base revision
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.base.sha",
					},
					Dest: "spec.arguments.parameters.6.value",
				},
				// Parameter 7: Set the base revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.base.ref",
					},
					Dest: "spec.arguments.parameters.7.value",
				},
				// Parameter 8: Set the pr number
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						// todo this is an int. Verify no side effects
						DataKey: "body.pull_request.number",
					},
					Dest: "spec.arguments.parameters.8.value",
				},
				// Append pull request number and short sha to dynamically assign workflow name <rollouts-demo-pr-21500-2c065a>
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataTemplate:   "{{ .Input.body.pull_request.number }}-{{ .Input.body.pull_request.head.sha | substr 0 7 }}-",
					},
					Dest:      "metadata.generateName",
					Operation: "append",
				},
			)
			workflowParameters = append(workflowParameters, prWorkflowParameters...)
		case *v1alpha1.GitHubPushTrigger:
			triggerParameters = append(
				triggerParameters,
				// Parameter 0: Set the repo url in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.repository.clone_url",
					},
					Dest: "spec.arguments.parameters.0.value",
				},
				// Parameter 1: Set the revision to check out
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.after",
					},
					Dest: "spec.arguments.parameters.1.value",
				},
				// Parameter 2: Set the repo full_name in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.repository.full_name",
					},
					Dest: "spec.arguments.parameters.2.value",
				},
				// Parameter 3: Set the revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.ref",
					},
					Dest: "spec.arguments.parameters.3.value",
				},
				// Parameter 4: Set the revision title
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						// todo message contains new lines. Need to strip
						DataTemplate: "{{ .Input.body.head_commit.message | splitList \"\\n\" | first | trunc 72 }}",
					},
					Dest: "spec.arguments.parameters.4.value",
				},
				// Parameter 5: Set the revision author
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pusher.name",
					},
					Dest: "spec.arguments.parameters.5.value",
				},
				// Append short sha to dynamically assign workflow name <rollouts-demo-commit-2c065a>
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataTemplate:   "{{ .Input.body.after | substr 0 7 }}-",
					},
					Dest:      "metadata.generateName",
					Operation: "append",
				},
			)
		default:
			return nil, nil, fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}
	return workflowParameters, triggerParameters, nil
}
