package webserver

import (
	"fmt"

	rolloutsv1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Same patch used by the argo-rollouts CLI `retry rollout` command:
// https://github.com/argoproj/argo-rollouts/blob/1b5bbd5bdd516e60acf3f93aa6c93a695464fdb3/pkg/kubectl-argo-rollouts/cmd/retry/retry.go#L89
const retryRolloutPatch = `{"status":{"abort":false}}`

func (s *FlowWatcher) handleRetryRolloutMessage(
	conn *WebConn,
	retryRolloutMessageData RetryRolloutMessageData,
) {
	conn.log.Info(
		"got retry rollout message",
		"namespace", retryRolloutMessageData.Namespace,
		"name", retryRolloutMessageData.Name,
	)

	go s.retryRollout(conn, retryRolloutMessageData)
}

func (s *FlowWatcher) retryRollout(
	conn *WebConn,
	retryRolloutMessageData RetryRolloutMessageData,
) {
	namespace := retryRolloutMessageData.Namespace
	name := retryRolloutMessageData.Name

	rollout := &rolloutsv1.Rollout{
		Namespace: namespace,
		Name:      name,
	}

	patch := ctrlclient.RawPatch(types.MergePatchType, []byte(retryRolloutPatch))
	err := s.client.Status().Patch(conn.ctx, rollout, patch)

	if err != nil {
		s.sendWebErrorAndClose(
			conn,
			err,
			namespace,
			name,
			rolloutRetryWebError,
			fmt.Sprintf("failed to retry rollout: %s/%s", namespace, name),
		)
		return
	}

	conn.log.Info(
		"rollout retried successfully",
		"name", name,
		"namespace", namespace,
	)
}
