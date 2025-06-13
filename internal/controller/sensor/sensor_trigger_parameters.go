package sensor

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
		{
			Name: "repo",
		},
		{
			Name: "revision",
		},
		{
			Name: "repo-short",
		},
	}
	// The PR workflow parameters. These will be available for each PR DAGTask to consume.
	// Each PR trigger is responsible for providing these via TriggerParameters
	prWorkflowParameters = []workflowsv1.Parameter{
		{
			Name: "revision-ref",
		},
		{
			Name: "base-revision",
		},
		{
			Name: "base-revision-ref",
		},
	}
	// The commit workflow parameters. These will be available for each commit DAGTask to consume.
	// Each commit trigger is responsible for providing these via TriggerParameters
	commitWorkflowParameters = []workflowsv1.Parameter{
		{
			Name: "revision-ref",
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
		len(globalWorkflowParameters)+len(prWorkflowParameters)+len(commitWorkflowParameters),
	)
	copy(workflowParameters, globalWorkflowParameters)

	var triggerParameters []eventsv1.TriggerParameter
	for i := range flowTriggers {
		switch trigger := flowTriggers[i].(type) {
		case *v1alpha1.GitHubPullRequestTrigger:
			triggerParameters = append(
				triggerParameters,
				// Set the repo url in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.repo.clone_url",
					},
					Dest: "spec.arguments.parameters.0.value",
				},
				// Set the revision to check out
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.sha",
					},
					Dest: "spec.arguments.parameters.1.value",
				},
				// Set the repo full_name in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.repo.full_name",
					},
					Dest: "spec.arguments.parameters.2.value",
				},
				// Set the revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.head.ref",
					},
					Dest: "spec.arguments.parameters.3.value",
				},
				// Set the base revision
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.base.sha",
					},
					Dest: "spec.arguments.parameters.4.value",
				},
				// Set the base revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.pull_request.base.ref",
					},
					Dest: "spec.arguments.parameters.5.value",
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
				// Set the repo url in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.repository.clone_url",
					},
					Dest: "spec.arguments.parameters.0.value",
				},
				// Set the revision to check out
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.after",
					},
					Dest: "spec.arguments.parameters.1.value",
				},
				// Set the repo full_name in the workflow
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.repository.full_name",
					},
					Dest: "spec.arguments.parameters.2.value",
				},
				// Set the revision ref
				eventsv1.TriggerParameter{
					Src: &eventsv1.TriggerParameterSource{
						DependencyName: "github-event-dep",
						DataKey:        "body.ref",
					},
					Dest: "spec.arguments.parameters.3.value",
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
			workflowParameters = append(workflowParameters, commitWorkflowParameters...)

		default:
			return nil, nil, fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}
	return workflowParameters, triggerParameters, nil
}
