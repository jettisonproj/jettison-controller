package webserver

import (
	"fmt"
)

func (s *FlowWatcher) handleSyncApplicationMessage(
	conn *WebConn,
	syncApplicationMessageData SyncApplicationMessageData,
) {
	namespace := syncApplicationMessageData.Namespace
	name := syncApplicationMessageData.Name

	err := s.syncClient.Sync(conn.ctx, namespace, name)
	if err != nil {
		s.sendWebErrorAndClose(
			conn,
			err,
			namespace,
			name,
			applicationSyncWebError,
			fmt.Sprintf("failed to sync ArgoCD application: %s/%s", namespace, name),
		)
		return
	}

	conn.log.Info(
		"application synced successfully in active conn",
		"name", name,
		"namespace", namespace,
	)
}
