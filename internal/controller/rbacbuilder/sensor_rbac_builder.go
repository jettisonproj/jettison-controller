package rbacbuilder

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Sensor RBAC based on:
// https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/sensor-rbac.yaml
//
// The Sensor needs RBAC to create workflows

const (
	// Name of the ServiceAccount used to run the Sensor pods
	SensorServiceAccountName = "operate-workflow-sa"

	sensorRoleName        = "operate-workflow-role"
	sensorRoleBindingName = "operate-workflow-role-binding"
)

func BuildSensorServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		Kind:       "ServiceAccount",
		APIVersion: corev1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       SensorServiceAccountName,
	}
}

func BuildSensorRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		Kind:       "Role",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       sensorRoleName,
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"argoproj.io"},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
				Resources: []string{
					"workflows",
				},
			},
		},
	}
}

func BuildSensorRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		Kind:       "RoleBinding",
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       sensorRoleBindingName,
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     sensorRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: SensorServiceAccountName,
			},
		},
	}
}
