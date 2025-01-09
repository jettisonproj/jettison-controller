package e2e

import (
	"fmt"

	"github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

const (
	flowKind  = "flows"
	namespace = "rollouts-demo"
)

// A client for Flow Custom Resources
type FlowClient struct {
	client *CommonClient
}

// Create a new flow given the yaml file path
func (c *FlowClient) Create(flowFilePath string) error {
	flowUrl := c.getFlowBaseUrl()
	err := c.client.Create(flowUrl, flowFilePath)

	if err != nil {
		return fmt.Errorf("failed to create flow: %s", err)
	}
	return nil
}

func (c *FlowClient) Delete(flowName string) error {
	flowUrl := c.getFlowUrl(flowName)
	err := c.client.Delete(flowUrl)
	if err != nil {
		return fmt.Errorf("failed to delete flow: %s", err)
	}
	return nil
}

func (c *FlowClient) Get(flowName string) (*v1alpha1.Flow, error) {
	flowUrl := c.getFlowUrl(flowName)

	// return err to propagate ErrorNotFound
	return CommonGet[v1alpha1.Flow](c.client, flowUrl)
}

func (c *FlowClient) getFlowUrl(flowName string) string {
	return fmt.Sprintf(
		"%s/%s",
		c.getFlowBaseUrl(),
		flowName,
	)
}

func (c *FlowClient) getFlowBaseUrl() string {
	return fmt.Sprintf(
		"/apis/%s/%s/namespaces/%s/%s",
		v1alpha1.GroupVersion.Group,
		v1alpha1.GroupVersion.Version,
		namespace,
		flowKind,
	)
}
