package sensor

import (
	"fmt"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

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
					Name:         "cicd-templates",
					Template:     "deploy-step-github-check-start",
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
					Name:         "cicd-templates",
					Template:     "deploy-step-github-check-start",
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

func getWorkflowTemplateDAGTasks(flowTriggers []v1alpha1base.BaseTrigger, flowSteps []v1alpha1base.BaseStep) ([]workflowsv1.DAGTask, string, error) {
	dagTasks, triggerType, err := getInitialDAGTasks(flowTriggers)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get initial dag tasks: %s", err)
	}

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
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         "cicd-templates",
					Template:     "docker-build-test",
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
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
							Name:  "image-destination",
							Value: workflowsv1.AnyStringPtr("{{inputs.parameters.repo-short}}"),
						},
					},
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         "cicd-templates",
					Template:     "docker-build-test-publish",
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
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
					Name:  "image-destination",
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
					Name:         "cicd-templates",
					Template:     "deploy-step-argocd",
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
		case *v1alpha1.DebugMessageStep:
			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
						{
							Name:  "message",
							Value: workflowsv1.AnyStringPtr(step.Message),
						},
					},
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         "cicd-templates",
					Template:     "whalesay",
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
					Name:         "cicd-templates",
					Template:     "approval",
					ClusterScope: true,
				},
				Dependencies: step.DependsOn,
			}
			dagTasks = append(dagTasks, dagTask)
		default:
			return nil, "", fmt.Errorf("unknown Flow step type: %T", step)
		}
	}
	return dagTasks, triggerType, nil
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
			Name:         "cicd-templates",
			Template:     "deploy-step-github-check-complete",
			ClusterScope: true,
		},
	}
}
