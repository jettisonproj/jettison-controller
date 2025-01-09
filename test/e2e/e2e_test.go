/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"os"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"
)

const (
	testdataDir = "../../testdata"
	kubeconfig  = "/etc/rancher/k3s/k3s.yaml"
)

func TestIntegrationGitHubPush(t *testing.T) {

	// create the custom resource client
	client, err := NewCrClient(kubeconfig)
	require.Nil(t, err, "failed to create CR client")

	// delete flow if it exists
	flowName := "github-push"
	_, err = client.Flow().Get(flowName)
	if err == nil {
		err = client.Flow().Delete(flowName)
		require.Nil(t, err, "failed to delete flow")

		_, err = client.Flow().Get(flowName)
	}

	// verify flow does not exist
	require.Truef(t, err != nil && errors.IsNotFound(err), "unexpected error getting flow: %s", err)

	// create flow
	flowFilePath := fmt.Sprintf("%s/%s", testdataDir, "github-push-minimal.yaml")
	err = client.Flow().Create(flowFilePath)
	require.Nil(t, err, "failed to create flow")

	// get expected sensor from file
	sensorFilePath := fmt.Sprintf("%s/%s", testdataDir, "github-push-minimal-sensor.yaml")
	sensorExpected, err := parseYaml[eventsv1.Sensor](sensorFilePath)
	require.Nilf(t, err, "failed to parse sensor %s", sensorFilePath)

	// get actual sensor from api
	sensorActual, err := client.Sensor().Get(flowName)
	require.Nilf(t, err, "failed to get sensor %s", flowName)

	// compare expected and actual sensors
	// first compare the workflow resource ([]byte field)
	actualWorkflowBytes := sensorActual.Spec.Triggers[0].Template.K8s.Source.Resource.Value
	var actualWorkflow workflowsv1.Workflow
	err = yaml.UnmarshalStrict(actualWorkflowBytes, &actualWorkflow)
	require.Nil(t, err, "failed to parse actual workflow")

	expectedWorkflowBytes := sensorExpected.Spec.Triggers[0].Template.K8s.Source.Resource.Value
	var expectedWorkflow workflowsv1.Workflow
	err = yaml.UnmarshalStrict(expectedWorkflowBytes, &expectedWorkflow)
	require.Nil(t, err, "failed to parse expected workflow")

	require.Equal(t, expectedWorkflow, actualWorkflow)

	// avoid comparing Kind and APIVersion. They are the same type anyway
	sensorActual.Kind = sensorExpected.Kind
	sensorActual.APIVersion = sensorExpected.APIVersion

	// avoid comparison with dynamically set fields
	sensorActual.Status = sensorExpected.Status
	sensorActual.OwnerReferences = sensorExpected.OwnerReferences
	sensorActual.ManagedFields = sensorExpected.ManagedFields
	sensorActual.Finalizers = sensorExpected.Finalizers
	sensorActual.Namespace = sensorExpected.Namespace
	sensorActual.UID = sensorExpected.UID
	sensorActual.ResourceVersion = sensorExpected.ResourceVersion
	sensorActual.Generation = sensorExpected.Generation
	sensorActual.CreationTimestamp = sensorExpected.CreationTimestamp
	sensorActual.DeletionTimestamp = sensorExpected.DeletionTimestamp
	sensorActual.DeletionGracePeriodSeconds = sensorExpected.DeletionGracePeriodSeconds

	// avoid comparing workflow resource ([]byte field) since it was compared above
	sensorActual.Spec.Triggers[0].Template.K8s.Source.Resource = nil
	sensorExpected.Spec.Triggers[0].Template.K8s.Source.Resource = nil

	require.Equal(t, sensorExpected, sensorActual)
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
