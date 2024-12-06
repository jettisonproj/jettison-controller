package base

// The common interface for triggers. Can typically be casted to the concrete type
// Ideally, this could be in the v1alpha1 package, but kubebuilder throws errors for interfaces
type BaseTrigger interface {
	// The name of trigger for the flow. This can be a description
	GetTriggerName() string
	// The type of trigger for the flow
	GetTriggerSource() string
}
