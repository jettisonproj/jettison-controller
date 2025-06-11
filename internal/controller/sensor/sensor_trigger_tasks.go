package sensor

import (
	"fmt"
	"path/filepath"
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

	stepsByName := make(map[string]v1alpha1base.BaseStep)
	for i := range flowSteps {
		stepsByName[flowSteps[i].GetStepName()] = flowSteps[i]
	}

	additionalTemplates := []workflowsv1.Template{}
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
				Depends: getDepends(step.DependsOn),
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
						{
							Name:  "dockerfile-dir",
							Value: workflowsv1.AnyStringPtr(getDockerfileDir(*step.DockerfilePath)),
						},
					},
				},
				Depends: getDepends(step.DependsOn),
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
			dockerfilePath, err := getDockerfilePathDependency(*step.StepName, stepsByName)
			if err != nil {
				return nil, nil, "", fmt.Errorf("error finding dockerfile dependency for step %s: %s", *step.StepName, err)
			}
			dockerfileDir := getDockerfileDir(dockerfilePath)

			dagTask := workflowsv1.DAGTask{
				Name: *step.StepName,
				Arguments: workflowsv1.Arguments{
					Parameters: []workflowsv1.Parameter{
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
						{
							Name:  "dockerfile-dir",
							Value: workflowsv1.AnyStringPtr(dockerfileDir),
						},
					},
				},
				TemplateRef: &workflowsv1.TemplateRef{
					Name:         workflowtemplates.CICDTemplate.ObjectMeta.Name,
					Template:     workflowtemplates.ArgoCDTemplate.Name,
					ClusterScope: true,
				},
				Depends: getDepends(step.DependsOn),
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

// Get the dockerfile dependency for the specified step.
// In the long term, there may be a more efficient solution
func getDockerfilePathDependency(initialStepName string, stepsByName map[string]v1alpha1base.BaseStep) (string, error) {
	initialStep := stepsByName[initialStepName]

	dependencies := make([]string, len(initialStep.GetDependsOn()))
	copy(dependencies, initialStep.GetDependsOn())

	visited := make(map[string]bool)
	dockerfilePath := ""
	for len(dependencies) > 0 {
		dependency := dependencies[0]
		dependencies = dependencies[1:]

		if !visited[dependency] {
			visited[dependency] = true
			switch step := stepsByName[dependency].(type) {
			case *v1alpha1.DockerBuildTestPublishStep:
				if dockerfilePath != "" {
					return "", fmt.Errorf("found multiple dockerfile paths for step: %s", initialStepName)
				}
				dockerfilePath = *step.DockerfilePath
			}
			dependencies = append(dependencies, stepsByName[dependency].GetDependsOn()...)
		}
	}

	if dockerfilePath == "" {
		return "", fmt.Errorf("did not find dockerfile path for step: %s", initialStepName)
	}

	return dockerfilePath, nil
}

// The dockerfileDir is used as a suffix in the image repo. The full image
// format is: <registry><repo-short><dockerfile-dir>:<revision>
func getDockerfileDir(dockerfilePath string) string {
	dir, file := filepath.Split(dockerfilePath)
	// Usually, file is simply "Dockerfile", but attempt to handle other conventions
	file = strings.ToLower(file)
	file = strings.ReplaceAll(file, "dockerfile", "")
	file = strings.Trim(file, " .-_")
	if file == "" && dir == "" {
		return ""
	}
	if dir == "" {
		return "/" + file
	}
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	if file == "" {
		return strings.TrimRight(dir, "/")
	}
	return dir + file
}

// Convert the "dependsOn" field set in the Flow to the "depends" field
// set in the Workflow. See:
// https://argo-workflows.readthedocs.io/en/latest/enhanced-depends-logic/
func getDepends(dependsOn []string) string {
	n := len(dependsOn)
	if n == 0 {
		return ""
	}

	succeededDependsOn := make([]string, n)
	for i, d := range dependsOn {
		succeededDependsOn[i] = fmt.Sprintf("%s.Succeeded", d)
	}

	return strings.Join(succeededDependsOn, " && ")
}
