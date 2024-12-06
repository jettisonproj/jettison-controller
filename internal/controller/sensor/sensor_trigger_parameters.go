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
)

func getTriggerParameters(flowTriggers []v1alpha1base.BaseTrigger) ([]eventsv1.TriggerParameter, error) {
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
			return nil, fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}
	return triggerParameters, nil
}
