package eventsources_test

import (
	"fmt"
	"os"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/eventsources"
)

const (
	testdataDir = "../../testdata"
)

func TestGetOwnedRepositories(t *testing.T) {
	prFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-pr-minimal.yaml")
	pushFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-push-minimal.yaml")
	prFlow, err := parseYaml[v1alpha1.Flow](prFlowPath)
	require.Nilf(t, err, "failed to parse pr flow: %s", prFlowPath)
	pushFlow, err := parseYaml[v1alpha1.Flow](pushFlowPath)
	require.Nilf(t, err, "failed to parse push flow: %s", pushFlowPath)

	flows := &v1alpha1.FlowList{
		Items: []v1alpha1.Flow{
			*prFlow,
			*pushFlow,
		},
	}

	ownedRepos, err := eventsources.GetOwnedRepositories(flows)
	require.Nilf(t, err, "failed get owned repos for flows")

	expectedOwnedRepos := []eventsv1.OwnedRepositories{
		{
			Owner: "jettisonproj",
			Names: []string{"rollouts-demo"},
		},
	}
	require.Equal(t, expectedOwnedRepos, ownedRepos)
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
