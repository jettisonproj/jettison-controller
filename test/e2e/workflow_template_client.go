package e2e

import (
	"fmt"

	"github.com/jettisonproj/jettison-controller/internal/workflowtemplates"

	workflows "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// A client for ClusterWorkflowTemplate Custom Resources
type WorkflowTemplateClient struct {
	client *CommonClient
}

func (c *WorkflowTemplateClient) Get() (*workflowsv1.ClusterWorkflowTemplate, error) {
	workflowTemplateUrl := c.getWorkflowTemplateUrl()

	// return err to propagate ErrorNotFound
	return CommonGet[workflowsv1.ClusterWorkflowTemplate](c.client, workflowTemplateUrl)
}

func (c *WorkflowTemplateClient) getWorkflowTemplateUrl() string {
	return fmt.Sprintf(
		"/apis/%s/%s/%s/%s",
		workflows.Group,
		workflows.Version,
		workflows.ClusterWorkflowTemplatePlural,
		workflowtemplates.CICDTemplate.ObjectMeta.Name,
	)
}
