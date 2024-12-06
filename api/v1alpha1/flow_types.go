/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FlowSpec defines the desired state of Flow.
type FlowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Optional deadline in seconds for running the flow.
	// Flows which miss the deadline will be counted as failed.
	// Defaults to 900 (15 minutes) for PRs and 3600 (1 hr) for pushes
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
	// List of triggers for running the flow.
	// See the concrete Trigger types in this package
	Triggers []RawMessage `json:"triggers"`
	// List of steps to run for each trigger
	// See the concrete Step types in this package
	Steps []RawMessage `json:"steps"`
}

// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:Type=object
type RawMessage struct {
	json.RawMessage `json:",inline"`
}

// FlowStatus defines the observed state of Flow.
type FlowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Represents the observations of a Flow's current state.
	// Flow.status.conditions.type are: "Ready"
	// Flow.status.conditions.status are one of True, False
	// Flow.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Flow.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Flow is the Schema for the flows API.
type Flow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowSpec   `json:"spec,omitempty"`
	Status FlowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlowList contains a list of Flow.
type FlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Flow{}, &FlowList{})
}
