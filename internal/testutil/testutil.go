package testutil

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// Parse the yaml file into a struct
func ParseYaml[T any](yamlFilePath string) (*T, error) {
	s := new(T)

	b, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return s, fmt.Errorf("failed to read file: %s", yamlFilePath)
	}
	err = yaml.UnmarshalStrict(b, s)
	if err != nil {
		return s, fmt.Errorf("parsing error: %s", err)
	}
	return s, nil
}
