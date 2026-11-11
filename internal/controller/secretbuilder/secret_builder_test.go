package secretbuilder_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/jettisonproj/jettison-controller/internal/controller/secretbuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func buildSourceSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		Namespace: secretbuilder.SourceNamespace,
		Name:      name,
		Annotations: map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": "{}",
		},
		Labels: map[string]string{
			"test-label": "test-value",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value-" + name),
		},
	}
}

func TestLoadFlowSecrets(t *testing.T) {
	var objects []client.Object
	for _, secretName := range secretbuilder.FlowSecretNames {
		objects = append(objects, buildSourceSecret(secretName))
	}
	// Secrets outside the source namespace are not loaded
	otherSecret := buildSourceSecret("github-access")
	otherSecret.Namespace = "other"
	otherSecret.Data["key"] = []byte("other")
	objects = append(objects, otherSecret)

	reader := fake.NewClientBuilder().WithObjects(objects...).Build()

	secrets, err := secretbuilder.LoadFlowSecrets(context.Background(), reader)
	require.Nil(t, err)
	require.Len(t, secrets, len(secretbuilder.FlowSecretNames))
	for i, secretName := range secretbuilder.FlowSecretNames {
		require.Equal(t, secretbuilder.SourceNamespace, secrets[i].Namespace)
		require.Equal(t, secretName, secrets[i].Name)
		require.Equal(t, []byte("value-"+secretName), secrets[i].Data["key"])
	}
}

func TestLoadFlowSecretsMissing(t *testing.T) {
	var objects []client.Object
	// Skip the last secret
	for _, secretName := range secretbuilder.FlowSecretNames[:len(secretbuilder.FlowSecretNames)-1] {
		objects = append(objects, buildSourceSecret(secretName))
	}

	reader := fake.NewClientBuilder().WithObjects(objects...).Build()

	secrets, err := secretbuilder.LoadFlowSecrets(context.Background(), reader)
	require.Nil(t, secrets)
	require.True(t, apierrors.IsNotFound(err), "expected not found error: %s", err)
	require.ErrorContains(t, err, "jettisonproj/s3-access")
}

func TestBuildFlowSecret(t *testing.T) {
	secretPath := fmt.Sprintf("%s/%s", testdataDir, "flow-secret.yaml")
	expectedSecret, err := testutil.ParseYaml[corev1.Secret](secretPath)
	require.Nilf(t, err, "failed to parse secret: %s", secretPath)

	sourceSecret := buildSourceSecret("github-access")
	secret := secretbuilder.BuildFlowSecret(sourceSecret, "wavesmp")

	require.Equal(t, expectedSecret, secret)

	// The data is copied, so changes to the copy do not affect the source
	secret.Data["key"] = []byte("changed")
	require.Equal(t, []byte("value-github-access"), sourceSecret.Data["key"])
}
