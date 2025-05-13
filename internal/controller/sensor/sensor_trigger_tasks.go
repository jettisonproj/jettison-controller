package sensor

import (
	"fmt"
	"strings"

	"github.com/jettisonproj/jettison-controller/internal/workflowtemplates"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

const (
	commitTriggerType = "commit"
	prTriggerType     = "PR"
)

func getInitialDAGTasks(flowTriggers []v1alpha1base.BaseTrigger) ([]workflowsv1.DAGTask, string, error) {
	var triggerType string
	var initialDAGTasks []workflowsv1.DAGTask
	for i := range flowTriggers {
		switch trigger := flowTriggers[i].(type) {
		case *v1alpha1.GitHubPullRequestTrigger:

			if triggerType == "" {
				triggerType = prTriggerType
			} else if triggerType != prTriggerType {
				return nil, "", fmt.Errorf("trigger types cannot be both %s and %s", prTriggerType, commitTriggerType)
			}

			initialDAGTask := workflowsv1.DAGTask{
				Name: "github-check-start",
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
						{
							Name:  "repo-short",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo-short}}"),
						},
						{
							Name:  "event-type",
							Value: workflowsv1.AnyStringPtr(triggerType),
						},
						{
							Name:  "revision",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.revision}}"),
						},
					},
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.GitHubCheckStartTemplate.Name,
					ClusterScope: true,
				},
			}
			initialDAGTasks = append(initialDAGTasks, initialDAGTask)

		case *v1alpha1.GitHubPushTrigger:

			if triggerType == "" {
				triggerType = commitTriggerType
			} else if triggerType != commitTriggerType {
				return nil, "", fmt.Errorf("trigger types cannot be both %s and %s", prTriggerType, commitTriggerType)
			}

			initialDAGTask := workflowsv1.DAGTask{
				Name: "github-check-start",
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
						{
							Name:  "repo-short",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo-short}}"),
						},
						{
							Name:  "event-type",
							Value: workflowsv1.AnyStringPtr(triggerType),
						},
						{
							Name:  "revision",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.revision}}"),
						},
					},
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.GitHubCheckStartTemplate.Name,
					ClusterScope: true,
				},
			}
			initialDAGTasks = append(initialDAGTasks, initialDAGTask)

		default:
			return nil, "", fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}
	return initialDAGTasks, triggerType, nil
}

func getWorkflowTemplateDAGTasks(flowTriggers []v1alpha1base.BaseTrigger, flowSteps []v1alpha1base.BaseStep) ([]workflowsv1.DAGTask, []workflowsv1.Template, string, error) {
	dagTasks, triggerType, err := getInitialDAGTasks(flowTriggers)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get initial dag tasks: %s", err)
	}

	additionalTemplates := []workflowsv1.Template{}
	manualApprovalSteps := map[string]bool{}
	for i := range flowSteps {
		switch step := flowSteps[i].(type) {
		case *v1alpha1.DockerBuildTestStep:
			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
						{
							Name:  "repo",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo}}"),
						},
						{
							Name:  "revision",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.revision}}"),
						},
						{
							Name:  "dockerfile-path",
							Value: workflowsv1.AnyStringPtr(*step.DockerfilePath),
						},
						{
							Name:  "docker-context-dir",
							Value: workflowsv1.AnyStringPtr(*step.DockerContextDir),
						},
					},
				},
				Dependencies: step.DependsOn,
			}

			if len(step.Volumes) == 0 && len(step.VolumeMounts) == 0 {
				dagTask.TemplateRef = &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.DockerBuildTestTemplate.Name,
					ClusterScope: true,
				}
				dagTasks = append(dagTasks, dagTask)
			} else {
				// From the Workflow, it is not possible to override the volume and
				// volumeMounts of the WorkflowTemplate, so generate the template here
				// See: https://github.com/argoproj/argo-workflows/issues/6677
				templateName := strings.ToLower(*step.StepName)

				template := workflowtemplates.DockerBuildTestTemplate
				template.Name = templateName

				templateContainerCopy := *template.Container
				template.Container = &templateContainerCopy

				template.Container.VolumeMounts = step.VolumeMounts
				template.Volumes = step.Volumes
				additionalTemplates = append(additionalTemplates, template)

				dagTask.Template = templateName
				dagTasks = append(dagTasks, dagTask)
			}
		case *v1alpha1.DockerBuildTestPublishStep:
			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
						{
							Name:  "repo",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo}}"),
						},
						{
							Name:  "revision",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.revision}}"),
						},
						{
							Name:  "dockerfile-path",
							Value: workflowsv1.AnyStringPtr(*step.DockerfilePath),
						},
						{
							Name:  "docker-context-dir",
							Value: workflowsv1.AnyStringPtr(*step.DockerContextDir),
						},
						{
							Name:  "image-repo",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo-short}}"),
						},
					},
				},
				Dependencies: step.DependsOn,
			}
			if len(step.Volumes) == 0 && len(step.VolumeMounts) == 0 {
				dagTask.TemplateRef = &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.DockerBuildTestPublishTemplate.Name,
					ClusterScope: true,
				}
				dagTasks = append(dagTasks, dagTask)
			} else {
				// From the Workflow, it is not possible to override the volume and
				// volumeMounts of the WorkflowTemplate, so generate the template here
				// This should be kept in sync with the cicd-templates
				// See: https://github.com/argoproj/argo-workflows/issues/6677
				templateName := strings.ToLower(*step.StepName)
				template := workflowtemplates.DockerBuildTestPublishTemplate
				template.Name = templateName

				templateVolumesCopy := make([]corev1.Volume, len(template.Volumes))
				copy(templateVolumesCopy, template.Volumes)
				template.Volumes = templateVolumesCopy

				template.Volumes = append(template.Volumes, step.Volumes...)

				templateContainerCopy := *template.Container
				template.Container = &templateContainerCopy

				template.Container.VolumeMounts = append(template.Container.VolumeMounts, step.VolumeMounts...)

				additionalTemplates = append(additionalTemplates, template)

				dagTask.Template = templateName
				dagTasks = append(dagTasks, dagTask)
			}
		case *v1alpha1.ArgoCDStep:
			parameters := []workflowsv1.Parameter{
				{
					Name:  "deploy-repo",
					Value: workflowsv1.AnyStringPtr(step.RepoUrl),
				},
				{
					Name:  "deploy-revision",
					Value: workflowsv1.AnyStringPtr(*step.BaseRef),
				},
				{
					Name:  "resource-path",
					Value: workflowsv1.AnyStringPtr(step.RepoPath),
				},
				{
					Name:  "image-repo",
					Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo-short}}"),
				},
				{
					Name:  "build-revision",
					Value: workflowsv1.AnyStringPtr("{{inputs.parameters.revision}}"),
				},
			}

			// If this ArgoCDStep depends on a ManualApprovalStep, then propagate
			// the approval status in order to pass/fail the deployment
			// Note that this relies on the step ordering. This may be ok since the
			// steps are usually in topological order
			for j := range step.DependsOn {
				if manualApprovalSteps[step.DependsOn[j]] {
					parameters = append(parameters, workflowsv1.Parameter{
						Name:  "deployment-approved",
						Value: workflowsv1.AnyStringPtr(fmt.Sprintf("{{tasks.%s.outputs.parameters.approve}}", step.DependsOn[j])),
					})
				}
			}

			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				Arguments: workflowsv1.Arguments{
					Parameters: parameters,
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.ArgoCDTemplate.Name,
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
		case *v1alpha1.ManualApprovalStep:
			manualApprovalSteps[*step.StepName] = true
			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.ApprovalTemplate.Name,
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
		default:
			return nil, nil, "", fmt.Errorf("unknown Flow step type: %T", step)
		}
	}
	return dagTasks, additionalTemplates, triggerType, nil
}

func getFinalDAGTask(triggerType string) workflowsv1.LifecycleHook {
	return workflowsv1.LifecycleHook{
		Arguments: workflowsv1.Arguments{
			Parameters: []workflowsv1.Parameter{
				{
					Name:  "repo-short",
					Value: workflowsv1.AnyStringPtr("{{workflow.parameters.repo-short}}"),
				},
				{
					Name:  "event-type",
					Value: workflowsv1.AnyStringPtr(triggerType),
				},
				{
					Name:  "check-run-id",
					Value: workflowsv1.AnyStringPtr("{{workflow.outputs.parameters.check-run-id}}"),
				},
				{
					Name:  "workflow-status",
					Value: workflowsv1.AnyStringPtr("{{workflow.status}}"),
				},
			},
		},
		TemplateRef: &workflowsv1.TemplateRef{
			Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
			Template:     workflowtemplates.GitHubCheckCompleteTemplate.Name,
			ClusterScope: true,
		},
	}
}
