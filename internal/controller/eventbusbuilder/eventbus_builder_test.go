package eventbusbuilder_test

import (
	"fmt"
	"testing"

	eventsv1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/stretchr/testify/require"

	"github.com/jettisonproj/jettison-controller/internal/controller/eventbusbuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildEventBus(t *testing.T) {
	eventBusPath := fmt.Sprintf("%s/%s", testdataDir, "eventbus-default.yaml")
	expectedEventBus, err := testutil.ParseYaml[eventsv1.EventBus](eventBusPath)
	require.Nilf(t, err, "failed to parse event bus: %s", eventBusPath)

	eventBus := eventbusbuilder.BuildEventBus("jettisonproj")

	require.Equal(t, expectedEventBus, eventBus)
}
