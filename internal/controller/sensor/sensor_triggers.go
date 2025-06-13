package sensor

import (
	"encoding/json"
	"fmt"

	eventscommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	workflows "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

// The Sensor trigger defines the actual k8s resources/workflows that will be launched
func getSensorTriggers(flow *v1alpha1.Flow, flowTriggers []v1alpha1base.BaseTrigger, flowSteps []v1alpha1base.BaseStep) ([]eventsv1.Trigger, error) {

	workflowParameters, triggerParameters, err := getTriggerParameters(flowTriggers)
	if err != nil {
		return nil, err
	}

	workflowTemplate, err := getWorkflowTemplate(flow, flowTriggers, flowSteps, workflowParameters)
	if err != nil {
		return nil, err
	}
	return []eventsv1.Trigger{
		{
			Template: &eventsv1.TriggerTemplate{
				Name: "flow-trigger",
				K8s: &eventsv1.StandardK8STrigger{
					Source: &eventsv1.ArtifactLocation{
						Resource: &eventscommon.Resource{
							Value: workflowTemplate,
						},
					},
					Operation:  eventsv1.Create,
					Parameters: triggerParameters,
				},
			},
		},
	}, nil
}

func getWorkflowTemplate(
	flow *v1alpha1.Flow,
	flowTriggers []v1alpha1base.BaseTrigger,
	flowSteps []v1alpha1base.BaseStep,
	workflowParameters []workflowsv1.Parameter,
) ([]byte, error) {
	flowWorkflowTemplateDAGTasks, additionalTemplates, triggerType, err := getWorkflowTemplateDAGTasks(flowTriggers, flowSteps)
	if err != nil {
		return nil, err
	}

	templates := []workflowsv1.Template{
		{
			Name: "main",
			Inputs: workflowsv1.Inputs{
				Parameters: workflowParameters,
			},
			DAG: &workflowsv1.DAGTemplate{
				Tasks: flowWorkflowTemplateDAGTasks,
			},
		},
	}

	if len(additionalTemplates) > 0 {
		templates = append(templates, additionalTemplates...)
	}

	workflowTemplate := workflowsv1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflows.WorkflowKind,
			APIVersion: workflows.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: flow.Name + "-",
		},
		Spec: workflowsv1.WorkflowSpec{
			Templates:  templates,
			Entrypoint: "main",
			Arguments: workflowsv1.Arguments{
				Parameters: workflowParameters,
			},
			// ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.
			// todo should extract out
			ServiceAccountName:    "deploy-step-executor",
			ActiveDeadlineSeconds: flow.Spec.ActiveDeadlineSeconds,
			Hooks: workflowsv1.LifecycleHooks{
				workflowsv1.ExitLifecycleEvent: getFinalDAGTask(triggerType),
			},
		},
	}

	workflowTemplateBytes, err := json.Marshal(workflowTemplate)
	if err != nil {
		return workflowTemplateBytes, fmt.Errorf("error marshaling workflow: %s", err)
	}
	return workflowTemplateBytes, nil
}
