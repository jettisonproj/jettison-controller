package pvcbuilder_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/jettisonproj/jettison-controller/internal/controller/pvcbuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildBuildKitCachePVC(t *testing.T) {
	pvcPath := fmt.Sprintf("%s/%s", testdataDir, "pvc-buildkit-cache.yaml")
	expectedPVC, err := testutil.ParseYaml[corev1.PersistentVolumeClaim](pvcPath)
	require.Nilf(t, err, "failed to parse pvc: %s", pvcPath)

	pvc := pvcbuilder.BuildBuildKitCachePVC("jettisonproj")

	require.Equal(t, expectedPVC, pvc)
}
