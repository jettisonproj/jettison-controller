package pvcbuilder

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// Name of the PVC used by the docker build deploy steps
	BuildKitCachePVCName = "buildkit-cache-pvc"

	buildKitCacheStorage = "15Gi"
)

func BuildBuildKitCachePVC(namespace string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		Kind:       "PersistentVolumeClaim",
		APIVersion: corev1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       BuildKitCachePVCName,
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(buildKitCacheStorage),
				},
			},
		},
	}
}
