package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

// Parse the triggers to concrete Trigger types from this package
func (f Flow) parseTriggers() ([]v1alpha1base.BaseTrigger, error) {
	triggers := make([]v1alpha1base.BaseTrigger, len(f.Spec.Triggers))
	for i := range f.Spec.Triggers {
		trigger, err := parseTrigger(f.Spec.Triggers[i])
		if err != nil {
			return triggers, err
		}
		triggers[i] = trigger
	}
	return triggers, nil
}

func parseTrigger(rawTrigger RawMessage) (v1alpha1base.BaseTrigger, error) {
	baseTrigger := BaseTriggerFields{}
	if err := json.Unmarshal(rawTrigger.RawMessage, &baseTrigger); err != nil {
		return nil, fmt.Errorf("failed to parse base trigger: %s", err)
	}

	switch baseTrigger.TriggerSource {
	case gitHubPullRequestTriggerSource:
		gitHubPullRequestTrigger := GitHubPullRequestTrigger{}
		if err := json.Unmarshal(rawTrigger.RawMessage, &gitHubPullRequestTrigger); err != nil {
			return nil, fmt.Errorf("failed to parse GitHubPullRequestTrigger: %s", err)
		}
		return &gitHubPullRequestTrigger, nil
	case gitHubPushTriggerSource:
		gitHubPushTrigger := GitHubPushTrigger{}
		if err := json.Unmarshal(rawTrigger.RawMessage, &gitHubPushTrigger); err != nil {
			return nil, fmt.Errorf("failed to parse GitHubPushTrigger: %s", err)
		}
		return &gitHubPushTrigger, nil
	default:
		return nil, fmt.Errorf("unknown triggerSource: %s should be one of: %s", baseTrigger.TriggerSource, strings.Join(triggerSources, ", "))
	}
}

// Parse the steps to concrete Step types from this package
func (f Flow) parseSteps() ([]v1alpha1base.BaseStep, error) {
	steps := make([]v1alpha1base.BaseStep, len(f.Spec.Steps))
	for i := range f.Spec.Steps {
		step, err := parseStep(f.Spec.Steps[i])
		if err != nil {
			return steps, err
		}
		steps[i] = step
	}
	return steps, nil
}

func parseStep(rawStep RawMessage) (v1alpha1base.BaseStep, error) {
	baseStep := BaseStepFields{}
	if err := json.Unmarshal(rawStep.RawMessage, &baseStep); err != nil {
		return nil, fmt.Errorf("failed to parse base step: %s", err)
	}

	switch baseStep.StepSource {
	case dockerBuildTestStepSource:
		dockerBuildTestStep := DockerBuildTestStep{}
		if err := json.Unmarshal(rawStep.RawMessage, &dockerBuildTestStep); err != nil {
			return nil, fmt.Errorf("failed to parse DockerBuildTestStep: %s", err)
		}
		return &dockerBuildTestStep, nil
	case dockerBuildTestPublishStepSource:
		dockerBuildTestPublishStep := DockerBuildTestPublishStep{}
		if err := json.Unmarshal(rawStep.RawMessage, &dockerBuildTestPublishStep); err != nil {
			return nil, fmt.Errorf("failed to parse DockerBuildTestPublishStep: %s", err)
		}
		return &dockerBuildTestPublishStep, nil
	case argoCDStepSource:
		argoCDStep := ArgoCDStep{}
		if err := json.Unmarshal(rawStep.RawMessage, &argoCDStep); err != nil {
			return nil, fmt.Errorf("failed to parse ArgoCDStep: %s", err)
		}
		return &argoCDStep, nil
	default:
		return nil, fmt.Errorf("unknown stepSource: %s should be one of: %s", baseStep.StepSource, strings.Join(stepSources, ", "))
	}
}
