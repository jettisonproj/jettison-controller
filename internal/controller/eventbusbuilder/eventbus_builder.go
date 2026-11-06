package eventbusbuilder

import (
	eventsv1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

const (
	eventBusName = "default"

	// NATS is expected to be configured externally
	jetStreamUrl = "nats://nats.nats"
)

func BuildEventBus(namespace string) *eventsv1.EventBus {
	return &eventsv1.EventBus{
		Kind:       eventsv1.EventBusGroupVersionKind.Kind,
		APIVersion: eventsv1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       eventBusName,
		Spec: eventsv1.EventBusSpec{
			JetStreamExotic: &eventsv1.JetStreamConfig{
				URL: jetStreamUrl,
			},
		},
	}
}
