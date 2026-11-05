package webserver

import (
	"encoding/json"
)

const (
	CONTAINER_LOG_MESSAGE_TYPE         = "containerLog"
	RESOURCE_SUBSCRIPTION_MESSAGE_TYPE = "resourceSubscription"
	SYNC_APPLICATION_MESSAGE_TYPE      = "syncApplication"
	RETRY_ROLLOUT_MESSAGE_TYPE         = "retryRollout"
)

type WebMessage struct {
	MessageType string          `json:"messageType"`
	MessageData json.RawMessage `json:"messageData"`
}

type ContainerLogMessageData struct {
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
}

type ResourceSubscription struct {
	SubscriptionType string `json:"subscriptionType"`
	Namespace        string `json:"namespace"`
	Name             string `json:"name"`
}

type SyncApplicationMessageData struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type RetryRolloutMessageData struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
