package webserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

const (
	webErrorKind = "WebError"
)

// WebError defines an error response for the webserver in a way that's consistent
// with Kubernetes resources
type WebErrorSpec struct {
	// The error message
	Message string `json:"message"`
}

// WebError is the Schema for the errors returned by the webserver
type WebError struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WebErrorSpec `json:"spec,omitempty"`
}

// WebErrorList contains a list of WebError.
type WebErrorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebError `json:"items"`
}

func newWebError(message string) WebErrorList {
	return WebErrorList{
		Items: []WebError{
			WebError{
				TypeMeta: metav1.TypeMeta{
					Kind:       webErrorKind,
					APIVersion: v1alpha1.GroupVersion.Identifier(),
				},
				Spec: WebErrorSpec{
					Message: message,
				},
			},
		},
	}
}
