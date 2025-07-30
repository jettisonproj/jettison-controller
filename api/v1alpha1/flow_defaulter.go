package v1alpha1

import (
	"fmt"

	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
)

const (
	defaultActiveDeadlineSecondsPR   int64 = 900
	defaultActiveDeadlineSecondsPush int64 = 3600
	defaultBaseRef                         = "main"
	defaultDockerContextDir                = ""

	// If the dockerContextDir is specified, the default is updated to:
	// <dockerContextDir>/Dockerfile
	defaultDockerfilePath = "Dockerfile"
)

var (
	defaultPullRequestEvents = []string{
		"opened",
		"reopened",
		"synchronize",
	}
)

// Apply default to the Flow, and the parsed triggers and steps
func (f *Flow) applyDefaults(triggers []v1alpha1base.BaseTrigger, steps []v1alpha1base.BaseStep) error {
	defaultActiveDeadlineSeconds := defaultActiveDeadlineSecondsPR
	for i := range triggers {
		switch trigger := triggers[i].(type) {
		case *GitHubPullRequestTrigger:
			if trigger.BaseRef == nil {
				trigger.BaseRef = new(string)
				*trigger.BaseRef = defaultBaseRef
			}

			if trigger.PullRequestEvents == nil {
				trigger.PullRequestEvents = defaultPullRequestEvents
			}
			defaultActiveDeadlineSeconds = max(defaultActiveDeadlineSeconds, defaultActiveDeadlineSecondsPR)
		case *GitHubPushTrigger:
			if trigger.BaseRef == nil {
				trigger.BaseRef = new(string)
				*trigger.BaseRef = defaultBaseRef
			}
			defaultActiveDeadlineSeconds = max(defaultActiveDeadlineSeconds, defaultActiveDeadlineSecondsPush)
		default:
			return fmt.Errorf("unknown trigger type: %T", trigger)
		}
	}

	if f.Spec.ActiveDeadlineSeconds == nil {
		f.Spec.ActiveDeadlineSeconds = new(int64)
		*f.Spec.ActiveDeadlineSeconds = defaultActiveDeadlineSeconds
	}

	for i := range steps {
		steps[i].ApplyDefaults()
		switch step := steps[i].(type) {
		case *DockerBuildTestStep:
			if step.DockerContextDir == nil {
				step.DockerContextDir = new(string)
				*step.DockerContextDir = defaultDockerContextDir
			}

			if step.DockerfilePath == nil {
				step.DockerfilePath = getDefaultDockerfilePath(*step.DockerContextDir)
			}
		case *DockerBuildTestPublishStep:
			if step.DockerContextDir == nil {
				step.DockerContextDir = new(string)
				*step.DockerContextDir = defaultDockerContextDir
			}

			if step.DockerfilePath == nil {
				step.DockerfilePath = getDefaultDockerfilePath(*step.DockerContextDir)
			}
		case *ArgoCDStep:
			if step.BaseRef == nil {
				step.BaseRef = new(string)
				*step.BaseRef = defaultBaseRef
			}
		default:
			return fmt.Errorf("unknown step type: %T", step)
		}
	}
	return nil
}

func getDefaultDockerfilePath(dockerContextDir string) *string {
	dockerfilePath := new(string)

	if dockerContextDir == "" {
		*dockerfilePath = defaultDockerfilePath
	} else {
		*dockerfilePath = fmt.Sprintf("%s/%s", dockerContextDir, defaultDockerfilePath)
	}

	return dockerfilePath
}
