package e2e

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// A client for custom resources (CRs)
// Uses kubernetes.Clientset
type CrClient struct {
	flowClient   *FlowClient
	sensorClient *SensorClient
}

// Create a new client using the specified kubeconfig path
func NewCrClient(kubeconfig string) (*CrClient, error) {
	masterUrl := "" // use the default

	config, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load from kubeconfig %s: %s", kubeconfig, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get clientset from config %s: %s", kubeconfig, err)
	}

	commonClient := &CommonClient{clientset}
	return &CrClient{
		flowClient:   &FlowClient{commonClient},
		sensorClient: &SensorClient{commonClient},
	}, nil
}

func (c *CrClient) Flow() *FlowClient {
	return c.flowClient
}

func (c *CrClient) Sensor() *SensorClient {
	return c.sensorClient
}
