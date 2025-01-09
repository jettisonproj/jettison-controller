package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type CommonClient struct {
	clientset *kubernetes.Clientset
}

// Create a new resource given the yaml file path
func (c *CommonClient) Create(createUrl string, filePath string) error {
	resourceBytes, err := c.yamlPathToJSON(filePath)
	if err != nil {
		return fmt.Errorf("failed to get json from file %s: %s", filePath, err)
	}

	resp, err := c.clientset.RESTClient().
		Post().
		AbsPath(createUrl).
		Body(resourceBytes).
		DoRaw(context.TODO())

	if err != nil {
		return fmt.Errorf(
			"failed to create resource with path=%s url=%s resp=%s: %s",
			filePath,
			createUrl,
			resp,
			err,
		)
	}
	return nil
}

func (c *CommonClient) Delete(deleteUrl string) error {
	resp, err := c.clientset.RESTClient().
		Delete().
		AbsPath(deleteUrl).
		DoRaw(context.TODO())

	if err != nil {
		return fmt.Errorf(
			"failed to delete resource with url=%s resp=%s: %s",
			deleteUrl,
			resp,
			err,
		)
	}
	return nil
}

// Ideally, if generics were supported on methods, we could return the
// appropriate type here
func (c *CommonClient) Get(getUrl string) ([]byte, error) {
	return c.clientset.RESTClient().Get().AbsPath(getUrl).DoRaw(context.TODO())
}

// Since generics are not supported on methods, use a function for the Get call
// as a workaround
func CommonGet[T any](c *CommonClient, getUrl string) (*T, error) {
	resourceBytes, err := c.Get(getUrl)
	if err != nil {
		// return err to propagate ErrNotFound
		return nil, err
	}

	resource := new(T)
	err = json.Unmarshal(resourceBytes, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource with url=%s: %s", getUrl, err)
	}
	return resource, nil
}

// Parse the yaml file into json bytes
func (c *CommonClient) yamlPathToJSON(yamlFilePath string) ([]byte, error) {
	inputBytes, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s", yamlFilePath)
	}
	outputBytes, err := yaml.YAMLToJSONStrict(inputBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting yaml path to json %s: %s", yamlFilePath, err)
	}
	return outputBytes, nil
}
