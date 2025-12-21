package webserver

import (
	"bufio"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

const (
	containerLogKind = "ContainerLog"
)

func (s *FlowWatcher) handleContainerLogMessage(
	conn *WebConn,
	containerLogMessageData ContainerLogMessageData,
) {
	conn.log.Info(
		"got container log message",
		"namespace", containerLogMessageData.Namespace,
		"podName", containerLogMessageData.PodName,
		"containerName", containerLogMessageData.ContainerName,
	)

	go s.streamLogLines(conn, containerLogMessageData)
	go s.sendPod(conn, containerLogMessageData)
}

// Based on:
// https://github.com/argoproj/argo-workflows/blob/main/util/logs/workflow-logger.go#L47
func (s *FlowWatcher) streamLogLines(
	conn *WebConn,
	containerLogMessageData ContainerLogMessageData,
) {
	podsInterface := s.kubeClient.Pods(containerLogMessageData.Namespace)

	podLogOptions := &v1.PodLogOptions{
		Container:  containerLogMessageData.ContainerName,
		Follow:     true,
		Timestamps: true,
	}
	ctx := conn.ctx
	stream, err := podsInterface.GetLogs(containerLogMessageData.PodName, podLogOptions).Stream(ctx)
	if err != nil {
		s.sendWebError(conn, err, "failed to get pod log stream")
		return
	}

	defer func() {
		if err := stream.Close(); err != nil {
			s.sendWebError(conn, err, "failed to close stream for pod logs")
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	logLines := []string{}

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendLogLines(conn, containerLogMessageData, logLines)
			logLines = logLines[:0]
		default:
			logLine := scanner.Text()
			logLines = append(logLines, logLine)
			conn.log.Info("got log message", "logLine", logLine)
		}
	}

	if err := scanner.Err(); err != nil {
		s.sendWebError(conn, err, "failed to scan pod log stream")
		return
	}

	if len(logLines) > 0 {
		s.sendLogLines(conn, containerLogMessageData, logLines)
	}

	conn.log.Info(
		"finished scanning log",
		"namespace", containerLogMessageData.Namespace,
		"podName", containerLogMessageData.PodName,
		"containerName", containerLogMessageData.ContainerName,
	)
}

func (s *FlowWatcher) sendLogLines(
	conn *WebConn,
	containerLogMessageData ContainerLogMessageData,
	logLines []string,
) {
	conn.log.Info("sending log lines", "numLines", len(logLines))
	s.notifyOne <- WebConnNotification{
		conn: conn,
		message: ContainerLogList{
			Items: []ContainerLog{
				ContainerLog{
					TypeMeta: metav1.TypeMeta{
						Kind:       containerLogKind,
						APIVersion: v1alpha1.GroupVersion.Identifier(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: containerLogMessageData.Namespace,
						Name:      containerLogMessageData.PodName,
					},
					Spec: ContainerLogSpec{
						ContainerName: containerLogMessageData.ContainerName,
						LogLines:      logLines,
					},
				},
			},
		},
	}
}

func (s *FlowWatcher) sendPod(
	conn *WebConn,
	containerLogMessageData ContainerLogMessageData,
) {
	podsInterface := s.kubeClient.Pods(containerLogMessageData.Namespace)
	pod, err := podsInterface.Get(
		conn.ctx,
		containerLogMessageData.PodName,
		metav1.GetOptions{},
	)
	if err != nil {
		msg := fmt.Sprintf(
			"failed to get pod %s/%s",
			containerLogMessageData.Namespace,
			containerLogMessageData.PodName,
		)
		s.sendWebError(conn, err, msg)
		return
	}
	if pod.APIVersion == "" || pod.Kind == "" {
		s.backfillPodSchema(pod)
	}
	s.notifyOne <- WebConnNotification{
		conn: conn,
		message: v1.PodList{
			Items: []v1.Pod{*pod},
		},
	}
}

// ContainerLogList contains a list of ContainerLog.
type ContainerLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerLog `json:"items"`
}

// ContainerLog is the Schema for the logs returned by the webserver
type ContainerLog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ContainerLogSpec `json:"spec,omitempty"`
}

// ContainerLog defines a log response for the webserver in a way that's consistent
// with Kubernetes resources
type ContainerLogSpec struct {
	// The container name
	ContainerName string `json:"containerName"`

	// The log messages
	LogLines []string `json:"logLines"`
}
