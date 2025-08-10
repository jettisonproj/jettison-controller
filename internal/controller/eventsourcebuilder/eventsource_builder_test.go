package eventsourcebuilder_test

import (
	"fmt"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/stretchr/testify/require"

	"github.com/jettisonproj/jettison-controller/internal/controller/eventsourcebuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildEventSource(t *testing.T) {
	eventSourcePath := fmt.Sprintf("%s/%s", testdataDir, "eventsource-github.yaml")
	expectedEventSource, err := testutil.ParseYaml[eventsv1.EventSource](eventSourcePath)
	require.Nilf(t, err, "failed to parse event source: %s", eventSourcePath)

	eventSource := eventsourcebuilder.BuildEventSource("jettisonproj", "rollouts-demo")

	require.Equal(t, expectedEventSource, eventSource)
}
