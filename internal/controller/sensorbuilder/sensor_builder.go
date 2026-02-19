package sensorbuilder

import (
	"fmt"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

// Build a Sensor for the specified Flow triggers and steps
func BuildSensor(
	flow *v1alpha1.Flow,
	flowTriggers []v1alpha1base.BaseTrigger,
	flowSteps []v1alpha1base.BaseStep,
) (*eventsv1.Sensor, error) {
	sensorDependencies, err := getSensorDependencies(flowTriggers)
	if err != nil {
		return nil, fmt.Errorf("error creating Sensor dependencies: %s", err)
	}

	sensorTriggers, err := getSensorTriggers(flow, flowTriggers, flowSteps)
	if err != nil {
		return nil, fmt.Errorf("error creating Sensor triggers: %s", err)
	}
	return &eventsv1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flow.ObjectMeta.Name,
			Namespace: flow.ObjectMeta.Namespace,
		},
		Spec: eventsv1.SensorSpec{
			Dependencies: sensorDependencies,
			Triggers:     sensorTriggers,
			Template: &eventsv1.Template{
				// ServiceAccountName is the name of the ServiceAccount to use to run sensor pod
				// todo should extract out
				ServiceAccountName: "operate-workflow-sa",
			},
		},
	}, nil
}
