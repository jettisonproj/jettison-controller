package sensor

import (
	"fmt"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

// The Flow triggers translate to Sensor event dependencies
func getSensorDependencies(flowTriggers []v1alpha1base.BaseTrigger) ([]eventsv1.EventDependency, error) {
	var sensorDependencies []eventsv1.EventDependency
	for i := range flowTriggers {
		switch trigger := flowTriggers[i].(type) {
		case *v1alpha1.GitHubPullRequestTrigger:
			sensorDependency := eventsv1.EventDependency{
				Name:            "github-event-dep",
				EventSourceName: "github",
				EventName:       "ghwebhook",
				Filters: &eventsv1.EventDependencyFilter{
					Data: []eventsv1.DataFilter{
						{
							Path:  "headers.X-Github-Event",
							Type:  eventsv1.JSONTypeString,
							Value: []string{"pull_request"},
						},
						{
							Path:  "body.action",
							Type:  eventsv1.JSONTypeString,
							Value: trigger.PullRequestEvents,
						},
						{
							Path:  "body.pull_request.state",
							Type:  eventsv1.JSONTypeString,
							Value: []string{"open"},
						},
						{
							Path:  "body.pull_request.base.ref",
							Type:  eventsv1.JSONTypeString,
							Value: []string{*trigger.BaseRef},
						},
					},
				},
			}
			sensorDependencies = append(sensorDependencies, sensorDependency)
		case *v1alpha1.GitHubPushTrigger:
			sensorDependency := eventsv1.EventDependency{
				Name:            "github-event-dep",
				EventSourceName: "github",
				EventName:       "ghwebhook",
				Filters: &eventsv1.EventDependencyFilter{
					Data: []eventsv1.DataFilter{
						{
							Path:  "headers.X-Github-Event",
							Type:  eventsv1.JSONTypeString,
							Value: []string{"push"},
						},
						{
							Path:  "body.ref",
							Type:  eventsv1.JSONTypeString,
							Value: []string{fmt.Sprintf("refs/heads/%s", *trigger.BaseRef)},
						},
					},
				},
			}
			sensorDependencies = append(sensorDependencies, sensorDependency)
		default:
			return nil, fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}
	return sensorDependencies, nil
}
