package e2e

import (
	"fmt"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

const (
	sensorKind = "sensors"
)

// A client for Sensor Custom Resources
type SensorClient struct {
	client *CommonClient
}

func (c *SensorClient) Get(flowName string) (*eventsv1.Sensor, error) {
	flowUrl := c.getSensorUrl(flowName)

	// return err to propagate ErrorNotFound
	return CommonGet[eventsv1.Sensor](c.client, flowUrl)
}

func (c *SensorClient) getSensorUrl(flowName string) string {
	return fmt.Sprintf(
		"%s/%s",
		c.getSensorBaseUrl(),
		flowName,
	)
}

func (c *SensorClient) getSensorBaseUrl() string {
	return fmt.Sprintf(
		"/apis/%s/%s/namespaces/%s/%s",
		eventsv1.SchemeGroupVersion.Group,
		eventsv1.SchemeGroupVersion.Version,
		namespace,
		sensorKind,
	)
}
