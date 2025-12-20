package webserver

import (
	"encoding/json"
)

const (
	CONTAINER_LOG_MESSAGE_TYPE = "containerLog"
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
