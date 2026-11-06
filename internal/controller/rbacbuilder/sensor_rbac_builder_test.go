package rbacbuilder_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/jettisonproj/jettison-controller/internal/controller/rbacbuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildSensorServiceAccount(t *testing.T) {
	serviceAccountPath := fmt.Sprintf("%s/%s", testdataDir, "sensor-rbac-serviceaccount.yaml")
	expectedServiceAccount, err := testutil.ParseYaml[corev1.ServiceAccount](serviceAccountPath)
	require.Nilf(t, err, "failed to parse service account: %s", serviceAccountPath)

	serviceAccount := rbacbuilder.BuildSensorServiceAccount("jettisonproj")

	require.Equal(t, expectedServiceAccount, serviceAccount)
}

func TestBuildSensorRole(t *testing.T) {
	rolePath := fmt.Sprintf("%s/%s", testdataDir, "sensor-rbac-role.yaml")
	expectedRole, err := testutil.ParseYaml[rbacv1.Role](rolePath)
	require.Nilf(t, err, "failed to parse role: %s", rolePath)

	role := rbacbuilder.BuildSensorRole("jettisonproj")

	require.Equal(t, expectedRole, role)
}

func TestBuildSensorRoleBinding(t *testing.T) {
	roleBindingPath := fmt.Sprintf("%s/%s", testdataDir, "sensor-rbac-rolebinding.yaml")
	expectedRoleBinding, err := testutil.ParseYaml[rbacv1.RoleBinding](roleBindingPath)
	require.Nilf(t, err, "failed to parse role binding: %s", roleBindingPath)

	roleBinding := rbacbuilder.BuildSensorRoleBinding("jettisonproj")

	require.Equal(t, expectedRoleBinding, roleBinding)
}
