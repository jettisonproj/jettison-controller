package webserver

import (
	"fmt"
)

const (
	POD_SUBSCRIPTION_TYPE          = "pod"
	ANALYSIS_RUN_SUBSCRIPTION_TYPE = "analysisRun"
	APP_INDEX_KEY                  = "metadata.labels.app"
	APP_LABEL_KEY                  = "app"
)

func (s *FlowWatcher) handleResourceSubscriptionMessage(
	conn *WebConn,
	resourceSubscription ResourceSubscription,
) {
	conn.log.Info(
		"got resource subscription message",
		"subscriptionType", resourceSubscription.SubscriptionType,
		"namespace", resourceSubscription.Namespace,
		"name", resourceSubscription.Name,
	)

	if resourceSubscription.SubscriptionType != POD_SUBSCRIPTION_TYPE && resourceSubscription.SubscriptionType != ANALYSIS_RUN_SUBSCRIPTION_TYPE {
		panic(fmt.Errorf("unexpected subscription type %s", resourceSubscription.SubscriptionType))
	}

	s.subscribe <- WebConnResourceSubscription{
		conn,
		resourceSubscription,
	}
}
