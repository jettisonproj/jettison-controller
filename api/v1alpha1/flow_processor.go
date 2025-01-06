package v1alpha1

import (
	"fmt"

	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

func (f *Flow) ProcessFlow() ([]v1alpha1base.BaseTrigger, []v1alpha1base.BaseStep, error) {
	flowTriggers, err := f.parseTriggers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to preprocess flow triggers: %s", err)
	}

	flowSteps, err := f.parseSteps()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to preprocess flow steps: %s", err)
	}

	if err := f.applyDefaults(flowTriggers, flowSteps); err != nil {
		return nil, nil, fmt.Errorf("failed to apply defaults to Flow: %s", err)
	}

	if err := f.validate(flowTriggers, flowSteps); err != nil {
		return nil, nil, fmt.Errorf("failed to validate Flow: %s", err)
	}

	return flowTriggers, flowSteps, nil
}
