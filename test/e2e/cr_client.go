package e2e

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// A client for custom resources (CRs)
// Uses kubernetes.Clientset
type CrClient struct {
	flowClient             *FlowClient
	sensorClient           *SensorClient
	workflowTemplateClient *WorkflowTemplateClient
}

// Create a new client using the kubeconfig (from env) or in-cluster config
func NewCrClient() (*CrClient, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes client config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get clientset from config: %s", err)
	}

	commonClient := &CommonClient{clientset}
	return &CrClient{
		flowClient:             &FlowClient{commonClient},
		sensorClient:           &SensorClient{commonClient},
		workflowTemplateClient: &WorkflowTemplateClient{commonClient},
	}, nil
}

func getConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		masterUrl := "" // use the default
		return clientcmd.BuildConfigFromFlags(masterUrl, kubeconfig)
	}
	return rest.InClusterConfig()
}

func (c *CrClient) Flow() *FlowClient {
	return c.flowClient
}

func (c *CrClient) Sensor() *SensorClient {
	return c.sensorClient
}

func (c *CrClient) WorkflowTemplate() *WorkflowTemplateClient {
	return c.workflowTemplateClient
}
