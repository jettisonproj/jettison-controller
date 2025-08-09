package workflowtemplates_test

import (
	"fmt"
	"testing"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"

	"github.com/jettisonproj/jettison-controller/internal/testutil"
	"github.com/jettisonproj/jettison-controller/internal/workflowtemplates"
)

const (
	testdataDir = "../../testdata"
)

func TestCICDTemplates(t *testing.T) {
	workflowTemplateFilePath := fmt.Sprintf("%s/%s", testdataDir, "cicd-templates.yaml")
	expectedWorkflowTemplate, err := testutil.ParseYaml[workflowsv1.ClusterWorkflowTemplate](workflowTemplateFilePath)
	require.Nilf(t, err, "failed to parse workflow template: %s", workflowTemplateFilePath)

	require.Equal(t, expectedWorkflowTemplate, &workflowtemplates.CICDTemplate)
}
