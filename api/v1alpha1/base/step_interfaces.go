package base

// The common interface for steps. Can typically be casted to the concrete type
// Ideally, this could be in the v1alpha1 package, but kubebuilder throws errors for interfaces
type BaseStep interface {
	// Optional name of step for the flow. Can be a description
	// Defaults to the StepSource
	GetStepName() string
	// The type of step for the flow
	GetStepSource() string
	// The names of steps which this step depends on
	GetDependsOn() []string
	// Apply defauts to the step. Usually happens when processing the step after parsing
	ApplyDefaults()
}
