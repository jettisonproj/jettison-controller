package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// The available step sources. Should be pascalcased
	dockerBuildTestStepSource        = "DockerBuildTest"
	dockerBuildTestPublishStepSource = "DockerBuildTestPublish"
	argoCDStepSource                 = "ArgoCD"
	debugMessageStepSource           = "DebugMessage"
	manualApprovalStepSource         = "ManualApproval"
)

var (
	stepSources = []string{
		dockerBuildTestStepSource,
		dockerBuildTestPublishStepSource,
		argoCDStepSource,
		debugMessageStepSource,
		manualApprovalStepSource,
	}
)

// The common json fields for steps
// Satisfies the BaseStep interface
type BaseStepFields struct {
	// Optional name of step for the flow. Can be a description
	// Defaults to the StepSource
	StepName *string `json:"stepName,omitempty"`
	// The type of step for the flow
	StepSource string `json:"stepSource"`
	// The names of steps which this step depends on
	DependsOn []string `json:"dependsOn,omitempty"`
	// Optional volumes to be used in the step container
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// Optional volume mounts for the step container
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

func (s BaseStepFields) GetStepName() string {
	return *s.StepName
}

func (s BaseStepFields) GetStepSource() string {
	return s.StepSource
}

func (s BaseStepFields) GetDependsOn() []string {
	return s.DependsOn
}

func (s *BaseStepFields) ApplyDefaults() {
	if s.StepName == nil {
		s.StepName = new(string)
		*s.StepName = s.StepSource
	}
}

type DockerBuildTestStep struct {
	BaseStepFields

	// Optional path to the Dockerfile that will be used for the build.
	// Defaults to "Dockerfile"
	// +optional
	DockerfilePath *string `json:"dockerfilePath,omitempty"`
	// Optional Docker context directory used for the build.
	// Defaults to "", which is the root of the repo
	// +optional
	DockerContextDir *string `json:"dockerContextDir,omitempty"`
}

type DockerBuildTestPublishStep struct {
	BaseStepFields

	// Optional path to the Dockerfile that will be used for the build.
	// Defaults to "Dockerfile"
	// +optional
	DockerfilePath *string `json:"dockerfilePath,omitempty"`
	// Optional Docker context directory used for the build.
	// Defaults to "", which is the root of the repo
	// +optional
	DockerContextDir *string `json:"dockerContextDir,omitempty"`
}

// Deploy using ArgoCD.
//
// Approvals (optional): If this stage depends on a `ManualApprovalStep`,
// and the approval status is 'NO', it will cause this step to fail
//
// Synchronization: For each combination of RepoUrl/RepoPath, only one
// step can run at a time
type ArgoCDStep struct {
	BaseStepFields

	// The url of the ArgoCD config repo. For example: https://github.com/osoriano/rollouts-demo-argo-configs.git
	// todo need to ensure this is in canonical format
	RepoUrl string `json:"repoUrl"`
	// The path of the k8s resources to update in the ArgoCD config repo.
	// This can be a directory such as dev, staging, prod. It can also be
	// an individual file path
	RepoPath string `json:"repoPath"`
	// Optional base ref for the push event. This is typically the default
	// branch name such as "main" or "master"
	// Defaults to "main"
	// +optional
	BaseRef *string `json:"baseRef,omitempty"`
}

type DebugMessageStep struct {
	BaseStepFields

	// The debug message to print
	Message string `json:"message"`
}

type ManualApprovalStep struct {
	BaseStepFields
}
