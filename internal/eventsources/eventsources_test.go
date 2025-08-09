package eventsources_test

import (
	"fmt"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/stretchr/testify/require"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/eventsources"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../testdata"
)

func TestGetOwnedRepositories(t *testing.T) {
	prFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-pr-minimal.yaml")
	pushFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-push-minimal.yaml")
	prFlow, err := testutil.ParseYaml[v1alpha1.Flow](prFlowPath)
	require.Nilf(t, err, "failed to parse pr flow: %s", prFlowPath)
	pushFlow, err := testutil.ParseYaml[v1alpha1.Flow](pushFlowPath)
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
