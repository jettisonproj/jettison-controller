package secretbuilder

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The Flow secrets are not stored in git. Instead, they are restored
// into the source namespace from a backup. Jettison loads them from the
// source namespace at startup and copies them into each Flow namespace

const (
	// Namespace containing the source of the Flow secrets
	SourceNamespace = "jettisonproj"
)

// Names of the secrets needed by each Flow namespace
var FlowSecretNames = []string{
	"github-access",
	"github-key",
	"regcred",
	"s3-access",
}

// Load the Flow secrets from the source namespace. Returns an error if
// any of the secrets fail to load
func LoadFlowSecrets(ctx context.Context, reader client.Reader) ([]*corev1.Secret, error) {
	secrets := make([]*corev1.Secret, 0, len(FlowSecretNames))
	for _, secretName := range FlowSecretNames {
		secret := &corev1.Secret{}
		secretKey := client.ObjectKey{
			Namespace: SourceNamespace,
			Name:      secretName,
		}
		if err := reader.Get(ctx, secretKey, secret); err != nil {
			return nil, fmt.Errorf("failed to load secret %s/%s: %w", SourceNamespace, secretName, err)
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

// Build a copy of the source secret for the specified namespace.
// Only the type and data are copied. The source metadata (e.g. the
// last-applied-configuration annotation) is not copied
func BuildFlowSecret(sourceSecret *corev1.Secret, namespace string) *corev1.Secret {
	return &corev1.Secret{
		Kind:       "Secret",
		APIVersion: corev1.SchemeGroupVersion.String(),
		Namespace:  namespace,
		Name:       sourceSecret.ObjectMeta.Name,
		Type:       sourceSecret.Type,
		Data:       maps.Clone(sourceSecret.Data),
	}
}
