package sensor

import (
	"fmt"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

func preProcessFlow(flow *v1alpha1.Flow) ([]v1alpha1base.BaseTrigger, []v1alpha1base.BaseStep, error) {
	flowTriggers, err := flow.ParseTriggers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to preprocess flow triggers: %s", err)
	}

	flowSteps, err := flow.ParseSteps()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to preprocess flow steps: %s", err)
	}

	if err := flow.ApplyDefaults(flowTriggers, flowSteps); err != nil {
		return nil, nil, fmt.Errorf("failed to apply defaults to Flow: %s", err)
	}

	if err := flow.Validate(flowTriggers, flowSteps); err != nil {
		return nil, nil, fmt.Errorf("failed to validate Flow: %s", err)
	}

	return flowTriggers, flowSteps, nil
}
