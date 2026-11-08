package rbacbuilder

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Workflow RBAC for the pods that run the Flow workflows
//
// The workflow pods need RBAC for:
//   - the Argo executor to report the workflowtaskresults
//   - the deploy steps to monitor resources after deploy (e.g. the ArgoCD deploy step)

const (
	// Name of the ServiceAccount used to run the workflow pods
	WorkflowServiceAccountName = "deploy-step-executor"

	// Name of the Role, RoleBinding, and ClusterRole. Also used as
	// the prefix of the per-namespace ClusterRoleBinding names
	WorkflowRBACName = "deploy-step-executor"
)

func BuildWorkflowServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		Kind:       "ServiceAccount",
		APIVersion: corev1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       WorkflowServiceAccountName,
	}
}

func BuildWorkflowRole(namespace string) *rbacv1.Role {
	readVerbs := []string{"get", "list", "watch"}
	return &rbacv1.Role{
		Kind:       "Role",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       WorkflowRBACName,
		Rules: []rbacv1.PolicyRule{
			// k8s standard APIs
			{
				APIGroups: []string{""},
				Resources: []string{"services", "serviceaccounts"},
				Verbs:     readVerbs,
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
				Verbs:     readVerbs,
			},
			{
				APIGroups: []string{rbacv1.GroupName},
				Resources: []string{"roles", "rolebindings"},
				Verbs:     readVerbs,
			},
			{
				APIGroups: []string{"gateway.networking.k8s.io"},
				Resources: []string{"httproutes"},
				Verbs:     readVerbs,
			},
			// Argo resources
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"analysistemplates", "rollouts"},
				Verbs:     readVerbs,
			},
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"workflowtaskresults"},
				Verbs:     []string{"create", "patch"},
			},
		},
	}
}

func BuildWorkflowRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		Kind:       "RoleBinding",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       WorkflowRBACName,
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     WorkflowRBACName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: WorkflowServiceAccountName,
			},
		},
	}
}

func BuildWorkflowClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		Kind:       "ClusterRole",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Name:       WorkflowRBACName,
		Rules: []rbacv1.PolicyRule{
			// k8s standard APIs
			{
				APIGroups: []string{rbacv1.GroupName},
				Resources: []string{"clusterroles", "clusterrolebindings"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

// Build the ClusterRoleBinding for the workflow ServiceAccount of the
// specified namespace. Each Flow namespace has its own ClusterRoleBinding,
// which is named using the namespace as a suffix
func BuildWorkflowClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		Kind:       "ClusterRoleBinding",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Name:       WorkflowClusterRoleBindingName(namespace),
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     WorkflowRBACName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      WorkflowServiceAccountName,
				Namespace: namespace,
			},
		},
	}
}

// Name of the ClusterRoleBinding for the specified Flow namespace
func WorkflowClusterRoleBindingName(namespace string) string {
	return WorkflowRBACName + "-" + namespace
}
