package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

// Parse the triggers to concrete Trigger types from this package
func (f Flow) ParseTriggers() ([]v1alpha1base.BaseTrigger, error) {
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

	switch normalizeSource(baseTrigger.TriggerSource) {
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
func (f Flow) ParseSteps() ([]v1alpha1base.BaseStep, error) {
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

	switch normalizeSource(baseStep.StepSource) {
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
	case debugMessageStepSource:
		debugMessageStep := DebugMessageStep{}
		if err := json.Unmarshal(rawStep.RawMessage, &debugMessageStep); err != nil {
			return nil, fmt.Errorf("failed to parse DebugMessageStep: %s", err)
		}
		return &debugMessageStep, nil
	case manualApprovalStepSource:
		manualApprovalStep := ManualApprovalStep{}
		if err := json.Unmarshal(rawStep.RawMessage, &manualApprovalStep); err != nil {
			return nil, fmt.Errorf("failed to parse ManualApprovalStep: %s", err)
		}
		return &manualApprovalStep, nil
	default:
		return nil, fmt.Errorf("unknown stepSource: %s should be one of: %s", baseStep.StepSource, strings.Join(stepSources, ", "))
	}
}

func normalizeSource(source string) string {
	charsToRemove := " .-_"
	for _, c := range charsToRemove {
		source = strings.ReplaceAll(source, string(c), "")
	}
	return strings.ToLower(source)
}
