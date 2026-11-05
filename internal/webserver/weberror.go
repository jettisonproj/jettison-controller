package webserver

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

const (
	webErrorKind = "WebError"

	flowListWebError        = "FlowListError"
	applicationListWebError = "ApplicationListError"
	rolloutListWebError     = "RolloutListError"
	workflowsListWebError   = "WorkflowListError"
	analysisRunListWebError = "AnalysisRunListError"
	podListWebError         = "PodListError"
	podLogsWebError         = "PodLogsError"
	podGetWebError          = "PodGetError"
	podNotFoundWebError     = "PodNotFound"
)

// WebError defines an error response for the webserver in a way that's consistent
// with Kubernetes resources
type WebErrorSpec struct {
	// The error reason
	Reason string `json:"reason"`

	// The error message
	Message string `json:"message"`
}

// WebError is the Schema for the errors returned by the webserver
type WebError struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec WebErrorSpec `json:"spec"`
}

// WebErrorList contains a list of WebError.
type WebErrorList struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []WebError `json:"items"`
}

func newWebError(
	namespace string,
	name string,
	reason string,
	message string,
) WebErrorList {
	return WebErrorList{
		Items: []WebError{
			{
				Kind:       webErrorKind,
				APIVersion: v1alpha1.GroupVersion.Identifier(),
				Namespace:  namespace,
				Name:       name,
				Spec: WebErrorSpec{
					Reason:  reason,
					Message: message,
				},
			},
		},
	}
}

func (s *FlowWatcher) sendWebErrorAndClose(
	conn *WebConn,
	err error,
	namespace string,
	name string,
	reason string,
	msg string,
) {
	conn.log.Error(err, msg)
	s.sendConn(conn, newWebError(
		namespace,
		name,
		reason,
		fmt.Sprintf("%s: %s", msg, err),
	))
	s.unregister <- conn
}

func (s *FlowWatcher) sendWebError(
	conn *WebConn,
	namespace string,
	name string,
	reason string,
	msg string,
) {
	s.sendConn(conn, newWebError(
		namespace,
		name,
		reason,
		msg,
	))
}
