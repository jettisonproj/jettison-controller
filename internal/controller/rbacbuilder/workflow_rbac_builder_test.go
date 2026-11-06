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

func TestBuildWorkflowServiceAccount(t *testing.T) {
	serviceAccountPath := fmt.Sprintf("%s/%s", testdataDir, "workflow-rbac-serviceaccount.yaml")
	expectedServiceAccount, err := testutil.ParseYaml[corev1.ServiceAccount](serviceAccountPath)
	require.Nilf(t, err, "failed to parse service account: %s", serviceAccountPath)

	serviceAccount := rbacbuilder.BuildWorkflowServiceAccount("jettisonproj")

	require.Equal(t, expectedServiceAccount, serviceAccount)
}

func TestBuildWorkflowRole(t *testing.T) {
	rolePath := fmt.Sprintf("%s/%s", testdataDir, "workflow-rbac-role.yaml")
	expectedRole, err := testutil.ParseYaml[rbacv1.Role](rolePath)
	require.Nilf(t, err, "failed to parse role: %s", rolePath)

	role := rbacbuilder.BuildWorkflowRole("jettisonproj")

	require.Equal(t, expectedRole, role)
}

func TestBuildWorkflowRoleBinding(t *testing.T) {
	roleBindingPath := fmt.Sprintf("%s/%s", testdataDir, "workflow-rbac-rolebinding.yaml")
	expectedRoleBinding, err := testutil.ParseYaml[rbacv1.RoleBinding](roleBindingPath)
	require.Nilf(t, err, "failed to parse role binding: %s", roleBindingPath)

	roleBinding := rbacbuilder.BuildWorkflowRoleBinding("jettisonproj")

	require.Equal(t, expectedRoleBinding, roleBinding)
}

func TestBuildWorkflowClusterRole(t *testing.T) {
	clusterRolePath := fmt.Sprintf("%s/%s", testdataDir, "workflow-rbac-clusterrole.yaml")
	expectedClusterRole, err := testutil.ParseYaml[rbacv1.ClusterRole](clusterRolePath)
	require.Nilf(t, err, "failed to parse cluster role: %s", clusterRolePath)

	clusterRole := rbacbuilder.BuildWorkflowClusterRole()

	require.Equal(t, expectedClusterRole, clusterRole)
}

func TestBuildWorkflowClusterRoleBinding(t *testing.T) {
	clusterRoleBindingPath := fmt.Sprintf("%s/%s", testdataDir, "workflow-rbac-clusterrolebinding.yaml")
	expectedClusterRoleBinding, err := testutil.ParseYaml[rbacv1.ClusterRoleBinding](clusterRoleBindingPath)
	require.Nilf(t, err, "failed to parse cluster role binding: %s", clusterRoleBindingPath)

	clusterRoleBinding := rbacbuilder.BuildWorkflowClusterRoleBinding("jettisonproj")

	require.Equal(t, expectedClusterRoleBinding, clusterRoleBinding)
}

func TestMergeWorkflowClusterRoleBindingSubject(t *testing.T) {
	clusterRoleBinding := rbacbuilder.BuildWorkflowClusterRoleBinding("jettisonproj")

	// The existing namespace is not added again
	updated := rbacbuilder.MergeWorkflowClusterRoleBindingSubject(clusterRoleBinding, "jettisonproj")
	require.False(t, updated)
	require.Len(t, clusterRoleBinding.Subjects, 1)

	// A new namespace is added
	updated = rbacbuilder.MergeWorkflowClusterRoleBindingSubject(clusterRoleBinding, "osoriano")
	require.True(t, updated)
	require.Equal(t, []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      rbacbuilder.WorkflowServiceAccountName,
			Namespace: "jettisonproj",
		},
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      rbacbuilder.WorkflowServiceAccountName,
			Namespace: "osoriano",
		},
	}, clusterRoleBinding.Subjects)

	// The new namespace is not added again
	updated = rbacbuilder.MergeWorkflowClusterRoleBindingSubject(clusterRoleBinding, "osoriano")
	require.False(t, updated)
	require.Len(t, clusterRoleBinding.Subjects, 2)
}
