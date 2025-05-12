package workflowtemplates_test

import (
	"fmt"
	"os"
	"testing"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/jettisonproj/jettison-controller/internal/workflowtemplates"
)

const (
	testdataDir = "../../testdata"
)

func TestCICDTemplates(t *testing.T) {
	workflowTemplateFilePath := fmt.Sprintf("%s/%s", testdataDir, "cicd-templates.yaml")
	expectedWorkflowTemplate, err := parseYaml[workflowsv1.ClusterWorkflowTemplate](workflowTemplateFilePath)
	require.Nilf(t, err, "failed to parse workflow template: %s", workflowTemplateFilePath)

	require.Equal(t, expectedWorkflowTemplate, &workflowtemplates.CICDTemplate)
}

// Parse the yaml file into a struct
func parseYaml[T any](yamlFilePath string) (*T, error) {
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
