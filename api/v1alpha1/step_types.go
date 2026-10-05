package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// The available step sources. Should be pascalcased
	dockerBuildTestStepSource        = "DockerBuildTest"
	dockerBuildTestPublishStepSource = "DockerBuildTestPublish"
	argoCDStepSource                 = "ArgoCD"
	gitHubCreatePrStepSource         = "GitHubCreatePR"
)

var (
	stepSources = []string{
		dockerBuildTestStepSource,
		dockerBuildTestPublishStepSource,
		argoCDStepSource,
		gitHubCreatePrStepSource,
	}
)

// The common json fields for steps
// Satisfies the BaseStep interface
type BaseStepFields struct {
	// Optional name of step for the flow. Can be a description
	// Defaults to the StepSource
	// +optional
	StepName *string `json:"stepName,omitzero"`
	// The type of step for the flow
	StepSource string `json:"stepSource"`
	// The names of steps which this step depends on
	// +optional
	DependsOn []string `json:"dependsOn,omitzero"`
	// Optional volumes to be used in the step container
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitzero"`
	// Optional volume mounts for the step container
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitzero"`
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
		s.StepName = new(s.StepSource)
	}
}

type DockerBuildArtifact struct {
	// The path to the build artifact
	Path string `json:"path"`
	// The key used for storing the build artifact
	Key string `json:"key"`
	// Mark the artifact as optional
	// By default, the step fails if an artifact path is not found.
	// If marked as optional, the step will instead continue without
	// saving the artifact
	// +optional
	Optional *bool `json:"optional,omitzero"`
}

type DockerBuildTestStep struct {
	BaseStepFields

	// Optional path to the Dockerfile that will be used for the build.
	// Defaults to "Dockerfile" or "<DockerContextDir>/Dockerfile"
	// if DockerContextDir is set
	// +optional
	DockerfilePath *string `json:"dockerfilePath,omitzero"`
	// Optional Docker context directory used for the build.
	// Defaults to "", which is the root of the repo
	// +optional
	DockerContextDir *string `json:"dockerContextDir,omitzero"`
	// Optional build artifacts to archive.
	// +optional
	Artifacts []DockerBuildArtifact `json:"artifacts,omitzero"`
}

type DockerBuildTestPublishStep struct {
	BaseStepFields

	// Optional path to the Dockerfile that will be used for the build.
	// Defaults to "Dockerfile" or "<DockerContextDir>/Dockerfile"
	// if DockerContextDir is set
	// +optional
	DockerfilePath *string `json:"dockerfilePath,omitzero"`
	// Optional Docker context directory used for the build.
	// Defaults to "", which is the root of the repo
	// +optional
	DockerContextDir *string `json:"dockerContextDir,omitzero"`
	// Optional build artifacts to archive.
	// +optional
	Artifacts []DockerBuildArtifact `json:"artifacts,omitzero"`
}

// Deploy using ArgoCD.
//
// Synchronization: For each combination of RepoUrl/RepoPath, only one
// step can run at a time
type ArgoCDStep struct {
	BaseStepFields

	// The url of the ArgoCD config repo. For example: https://github.com/jettisonproj/rollouts-demo-argo-configs.git
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
	BaseRef *string `json:"baseRef,omitzero"`
	// Optional, if non-blank, this effectively pauses the step
	// by disabling the automated syncs
	// +optional
	PauseReason *string `json:"pauseReason,omitzero"`
}

// Create GitHub PR after substituting the image tag in the specified repo and files
type GitHubCreatePrStep struct {
	BaseStepFields

	// The url of the repo to update. For example: https://github.com/jettisonproj/deploy-steps.git
	// todo need to ensure this is in canonical format
	RepoUrl string `json:"repoUrl"`
	// Optional base ref of the repo to update. This is typically the default
	// branch name such as "main" or "master"
	// Defaults to "main"
	// +optional
	BaseRef *string `json:"baseRef,omitzero"`
	// The file paths to substitute the image tags in
	FilePaths []string `json:"filePaths"`
}
