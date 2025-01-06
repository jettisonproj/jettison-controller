package v1alpha1

import (
	"fmt"

	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

func (f Flow) validate(triggers []v1alpha1base.BaseTrigger, steps []v1alpha1base.BaseStep) error {
	if len(triggers) == 0 {
		return fmt.Errorf("triggers cannot be empty")
	}

	for i := range triggers {

		switch trigger := triggers[i].(type) {
		case *GitHubPullRequestTrigger:
			if len(trigger.PullRequestEvents) == 0 {
				return fmt.Errorf("GitHubPullRequestTrigger pullRequestEvents cannot be empty")
			}
		case *GitHubPushTrigger:
			// No additional validation needed
		default:
			return fmt.Errorf("unknown trigger type: %T", trigger)

		}
	}

	if len(steps) == 0 {
		return fmt.Errorf("steps cannot be empty")
	}

	return nil
}
