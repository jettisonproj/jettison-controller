package sensor_test

import (
	"fmt"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/controller/sensor"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildSensor(t *testing.T) {
	testCases := []struct {
		flowFilePath   string
		sensorFilePath string
	}{
		{
			"github-pr-minimal.yaml",
			"github-pr-minimal-sensor.yaml",
		},
		{
			"github-pr-maximal.yaml",
			"github-pr-maximal-sensor.yaml",
		},
		{
			"github-push-minimal.yaml",
			"github-push-minimal-sensor.yaml",
		},
		{
			"github-push-maximal.yaml",
			"github-push-maximal-sensor.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.flowFilePath, func(t *testing.T) {
			flowFilePath := fmt.Sprintf("%s/%s", testdataDir, tc.flowFilePath)
			flow, err := testutil.ParseYaml[v1alpha1.Flow](flowFilePath)
			require.Nilf(t, err, "failed to parse flow: %s", flowFilePath)

			sensorActual, err := sensor.BuildSensor(flow)
			require.Nil(t, err, "failed to build sensor")

			sensorFilePath := fmt.Sprintf("%s/%s", testdataDir, tc.sensorFilePath)
			sensorExpected, err := testutil.ParseYaml[eventsv1.Sensor](sensorFilePath)
			require.Nil(t, err, "failed to parse sensor")

			// First compare the workflow resource ([]byte field)
			actualWorkflowBytes := sensorActual.Spec.Triggers[0].Template.K8s.Source.Resource.Value
			var actualWorkflow workflowsv1.Workflow
			err = yaml.UnmarshalStrict(actualWorkflowBytes, &actualWorkflow)
			require.Nil(t, err, "failed to parse actual workflow")

			expectedWorkflowBytes := sensorExpected.Spec.Triggers[0].Template.K8s.Source.Resource.Value
			var expectedWorkflow workflowsv1.Workflow
			err = yaml.UnmarshalStrict(expectedWorkflowBytes, &expectedWorkflow)
			require.Nil(t, err, "failed to parse expected workflow")

			require.Equal(t, expectedWorkflow, actualWorkflow)

			// Avoid comparing Kind and APIVersion. They are the same type anyway
			sensorActual.Kind = sensorExpected.Kind
			sensorActual.APIVersion = sensorExpected.APIVersion

			// Avoid comparing workflow resource ([]byte field) since it was compared above
			sensorActual.Spec.Triggers[0].Template.K8s.Source.Resource = nil
			sensorExpected.Spec.Triggers[0].Template.K8s.Source.Resource = nil

			require.Equal(t, sensorExpected, sensorActual)
		})
	}
}
