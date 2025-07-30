package v1alpha1_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

const (
	testdataDir = "../../testdata"
)

func TestFlowPR(t *testing.T) {
	testCases := []struct {
		flowFilePath                  string
		expectedActiveDeadlineSeconds int64
		expectedBaseRef               string
		expectedPullRequestEvents     []string
		expectedDockerfilePath        string
		expectedDockerfileContextDir  string
		expectedVolumesLen            int
		expectedVolumeMountsLen       int
	}{
		{
			// For the minimal yaml, defaults are expected
			"github-pr-minimal.yaml",
			900,
			"main",
			[]string{"opened", "reopened", "synchronize"},
			"Dockerfile",
			"",
			0,
			0,
		},
		{
			// For the maximal yaml, overrides are expected
			"github-pr-maximal.yaml",
			800,
			"master",
			[]string{"opened"},
			"subdir/Dockerfile",
			"subdir",
			1,
			1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.flowFilePath, func(t *testing.T) {

			yamlFilePath := fmt.Sprintf("%s/%s", testdataDir, tc.flowFilePath)
			flow, err := parseYaml[v1alpha1.Flow](yamlFilePath)
			require.Nilf(t, err, "failed to parse flow: %s", tc.flowFilePath)

			flowTriggers, flowSteps, err := flow.ProcessFlow()
			require.Nil(t, err, "failed to process flow")

			require.Equal(t, "workflows.jettisonproj.io/v1alpha1", flow.TypeMeta.APIVersion)
			require.Equal(t, "Flow", flow.TypeMeta.Kind)
			require.Equal(t, "github-pr", flow.ObjectMeta.Name)

			require.Equal(t, tc.expectedActiveDeadlineSeconds, *flow.Spec.ActiveDeadlineSeconds)

			require.Len(t, flowTriggers, 1)
			flowTrigger := flowTriggers[0]
			require.Equal(t, "GitHubPullRequest", flowTrigger.GetTriggerSource())

			prTrigger, ok := flowTrigger.(*v1alpha1.GitHubPullRequestTrigger)
			require.Truef(t, ok, "failed to parse type as *GitHubPullRequestTrigger: %T", flowTrigger)
			require.Equal(t, "https://github.com/jettisonproj/rollouts-demo.git", prTrigger.RepoUrl)
			require.Equal(t, tc.expectedBaseRef, *prTrigger.BaseRef)
			require.Equal(t, tc.expectedPullRequestEvents, prTrigger.PullRequestEvents)

			require.Len(t, flowSteps, 1)
			flowStep := flowSteps[0]
			require.Equal(t, "DockerBuildTest", flowStep.GetStepSource())
			require.Equal(t, "DockerBuildTest", flowStep.GetStepName())

			buildStep, ok := flowStep.(*v1alpha1.DockerBuildTestStep)
			require.Truef(t, ok, "failed to parse step as *DockerBuildTestStep: %T", flowStep)
			require.Equal(t, tc.expectedDockerfilePath, *buildStep.DockerfilePath)
			require.Equal(t, tc.expectedDockerfileContextDir, *buildStep.DockerContextDir)
			require.Equal(t, tc.expectedVolumesLen, len(buildStep.Volumes))
			require.Equal(t, tc.expectedVolumeMountsLen, len(buildStep.VolumeMounts))
		})
	}
}

func TestFlowPush(t *testing.T) {
	testCases := []struct {
		flowFilePath                  string
		expectedActiveDeadlineSeconds int64
		expectedBaseRef               string
		expectedDockerfilePath        string
		expectedDockerfileContextDir  string
		expectedVolumesLen            int
		expectedVolumeMountsLen       int
	}{
		{
			// For the minimal yaml, defaults are expected
			"github-push-minimal.yaml",
			3600,
			"main",
			"Dockerfile",
			"",
			0,
			0,
		},
		{
			// For the maximal yaml, overrides are expected
			"github-push-maximal.yaml",
			7200,
			"master",
			"subdir/Dockerfile",
			"subdir",
			1,
			1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.flowFilePath, func(t *testing.T) {

			yamlFilePath := fmt.Sprintf("%s/%s", testdataDir, tc.flowFilePath)
			flow, err := parseYaml[v1alpha1.Flow](yamlFilePath)
			require.Nilf(t, err, "failed to parse flow: %s", tc.flowFilePath)

			flowTriggers, flowSteps, err := flow.ProcessFlow()
			require.Nil(t, err, "failed to process flow")

			require.Equal(t, "workflows.jettisonproj.io/v1alpha1", flow.TypeMeta.APIVersion)
			require.Equal(t, "Flow", flow.TypeMeta.Kind)
			require.Equal(t, "github-push", flow.ObjectMeta.Name)

			require.Equal(t, tc.expectedActiveDeadlineSeconds, *flow.Spec.ActiveDeadlineSeconds)

			require.Len(t, flowTriggers, 1)
			flowTrigger := flowTriggers[0]
			require.Equal(t, "github-push", flowTrigger.GetTriggerName())
			require.Equal(t, "GitHubPush", flowTrigger.GetTriggerSource())

			pushTrigger, ok := flowTrigger.(*v1alpha1.GitHubPushTrigger)
			require.Truef(t, ok, "failed to parse type as *GitHubPushTrigger: %T", flowTrigger)
			require.Equal(t, "https://github.com/jettisonproj/rollouts-demo.git", pushTrigger.RepoUrl)
			require.Equal(t, tc.expectedBaseRef, *pushTrigger.BaseRef)

			require.Len(t, flowSteps, 4)

			// Step 0 - Build
			require.Equal(t, "docker-build-test-publish", flowSteps[0].GetStepName())
			require.Equal(t, "DockerBuildTestPublish", flowSteps[0].GetStepSource())
			require.Empty(t, flowSteps[0].GetDependsOn())

			buildStep, ok := flowSteps[0].(*v1alpha1.DockerBuildTestPublishStep)
			require.Truef(t, ok, "failed to parse step 0 as *DockerBuildTestPublishStep: %T", flowSteps[0])
			require.Equal(t, tc.expectedDockerfilePath, *buildStep.DockerfilePath)
			require.Equal(t, tc.expectedDockerfileContextDir, *buildStep.DockerContextDir)
			require.Equal(t, tc.expectedVolumesLen, len(buildStep.Volumes))
			require.Equal(t, tc.expectedVolumeMountsLen, len(buildStep.VolumeMounts))

			// Step 1 - Dev
			require.Equal(t, "deploy-to-dev", flowSteps[1].GetStepName())
			require.Equal(t, "ArgoCD", flowSteps[1].GetStepSource())
			require.Equal(t, []string{"docker-build-test-publish"}, flowSteps[1].GetDependsOn())

			devStep, ok := flowSteps[1].(*v1alpha1.ArgoCDStep)
			require.Truef(t, ok, "failed to parse step 1 as *ArgoCDStep: %T", flowSteps[1])
			require.Equal(t, "https://github.com/jettisonproj/rollouts-demo-argo-configs.git", devStep.RepoUrl)
			require.Equal(t, "dev", devStep.RepoPath)
			require.Equal(t, tc.expectedBaseRef, *devStep.BaseRef)

			// Step 2 - Staging
			require.Equal(t, "deploy-to-staging", flowSteps[2].GetStepName())
			require.Equal(t, "ArgoCD", flowSteps[2].GetStepSource())
			require.Equal(t, []string{"docker-build-test-publish"}, flowSteps[2].GetDependsOn())

			stagingStep, ok := flowSteps[2].(*v1alpha1.ArgoCDStep)
			require.Truef(t, ok, "failed to parse step 2 as *ArgoCDStep: %T", flowSteps[2])
			require.Equal(t, "https://github.com/jettisonproj/rollouts-demo-argo-configs.git", stagingStep.RepoUrl)
			require.Equal(t, "staging", stagingStep.RepoPath)
			require.Equal(t, tc.expectedBaseRef, *stagingStep.BaseRef)

			// Step 3 - Prod
			require.Equal(t, "deploy-to-prod", flowSteps[3].GetStepName())
			require.Equal(t, "ArgoCD", flowSteps[3].GetStepSource())
			require.Equal(t, []string{"deploy-to-staging"}, flowSteps[3].GetDependsOn())

			prodStep, ok := flowSteps[3].(*v1alpha1.ArgoCDStep)
			require.Truef(t, ok, "failed to parse step 3 as *ArgoCDStep: %T", flowSteps[3])
			require.Equal(t, "https://github.com/jettisonproj/rollouts-demo-argo-configs.git", prodStep.RepoUrl)
			require.Equal(t, "prod", prodStep.RepoPath)
			require.Equal(t, tc.expectedBaseRef, *prodStep.BaseRef)

		})
	}
}

// Parse the yaml file into a struct
func parseYaml[T any](yamlFilePath string) (*T, error) {
	s := new(T)

	b, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return s, fmt.Errorf("failed to read file: %s", yamlFilePath)
	}
	err = yaml.UnmarshalStrict(b, s)
	if err != nil {
		return s, fmt.Errorf("parsing error: %s", err)
	}
	return s, nil
}
