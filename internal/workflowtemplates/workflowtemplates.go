package workflowtemplates

import (
	"context"
	"time"

	workflows "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The image registry used for pushing images. Set to blank to use docker hub
	imageRegistry = "ghcr.io/"
	// GitHub App ID of the jettisonproj app
	githubAppId = "1682308"
	// GitHub App User ID of the jettisonproj app
	githubAppUserId = "223206062"
	// GitHub App User Name of jettisonproj app
	githubAppUserName = "jettisonproj[bot]"
	// Deploy Step Image for GitHub Checks
	deployStepsGitHubCheckImage = "ghcr.io/jettisonproj/deploy-steps/github-check:a8043f90be5a8330782173647ea245fba308ac94"
	// Deploy Step Image for ArgoCD Config Update
	deployStepsArgoCDImage = "ghcr.io/jettisonproj/deploy-steps/argocd:d20af9df44dd9fe0f315c9c0620a229e59d60de9"
	// Deploy Step Image for Docker Build Diff Check
	deployStepsDockerBuildDiffCheckImage = "ghcr.io/jettisonproj/deploy-steps/docker-build-diff-check:1e1103b7308cf97af3bdd44747743dac507e210b"
	// Deploy Step Image for Docker Build
	deployStepsDockerBuildImage = "ghcr.io/jettisonproj/deploy-steps/docker-build:da9f01d7adad4beb879ba3f50c3d7791ebf902b7"
)

var (
	log = ctrl.Log.WithName("workflowtemplates")

	activeDeadlineSeconds5m = intstr.FromInt(300)    // 5m
	activeDeadlineSeconds3d = intstr.FromInt(259200) // 3d

	// deploy-step-github-check-start
	// Starts a GitHub check for the specified commit.
	// Outputs the check run id, so it can be completed later
	GitHubCheckStartTemplate = workflowsv1.Template{
		Name:                  "deploy-step-github-check-start",
		ActiveDeadlineSeconds: &activeDeadlineSeconds5m, // 5m
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				// github-app-id - app id used for updating GitHub checks
				{
					Name:  "github-app-id",
					Value: workflowsv1.AnyStringPtr(githubAppId),
				},
				// repo-short - repo in the format <org-name>/<repo-name>
				{
					Name: "repo-short",
				},
				// workflow-url - the url to the Argo Workflow UI of this workflow run
				{
					Name:  "workflow-url",
					Value: workflowsv1.AnyStringPtr("https://argo.osoriano.com/workflows/{{workflow.namespace}}/{{workflow.name}}"),
				},
				// event-type - a short description of the event type (e.g. PR or Commit)
				{
					Name: "event-type",
				},
				// revision - the commit sha
				{
					Name: "revision",
				},
			},
		},
		Container: &corev1.Container{
			Image: deployStepsGitHubCheckImage,
			Args: []string{
				"./github-check-start.sh",
				"{{inputs.parameters.github-app-id}}",
				"/github-key/private-key.pem",
				"{{inputs.parameters.repo-short}}",
				"{{inputs.parameters.workflow-url}}",
				"{{inputs.parameters.event-type}}",
				"{{inputs.parameters.revision}}",
				"/tmp/check-run-id.txt",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					// Mount the configuration so we can update the git status
					Name:      "github-key",
					MountPath: "/github-key",
				},
			},
		},
		Volumes: []corev1.Volume{
			// Mount the configuration so we can update the git status
			{
				Name: "github-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "github-key",
					},
				},
			},
		},
		// export a global parameter so that it's available in the global exit handler.
		// it can be accessed under the object: workflow.outputs.parameters
		Outputs: workflowsv1.Outputs{
			Parameters: []workflowsv1.Parameter{
				{
					Name: "check-run-id",
					ValueFrom: &workflowsv1.ValueFrom{
						Path: "/tmp/check-run-id.txt",
					},
					GlobalName: "check-run-id",
				},
			},
		},
	}

	// docker-build-test
	// Useful for PR triggers. Will run the image build, but skips publishing it
	DockerBuildTestTemplate = workflowsv1.Template{
		Name: "docker-build-test",
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				//  repo - the repo that will be cloned
				{
					Name: "repo",
				},
				// revision - the revision or commit sha that will be checked out
				{
					Name: "revision",
				},
				// dockerfile-path - the path to the Dockerfile which will be built
				{
					Name: "dockerfile-path",
				},
				// docker-context-dir - the path to the docker context dir that will be
				//                      used for the build
				{
					Name: "docker-context-dir",
				},
				// revision-ref - the revision ref that will be used locally
				{
					Name: "revision-ref",
				},
				// base-revision - the base revision or commit sha for the change
				{
					Name: "base-revision",
				},
				// base-revision-ref - the base revision ref that will be used locally
				{
					Name: "base-revision-ref",
				},
			},
		},
		ContainerSet: &workflowsv1.ContainerSetTemplate{
			Containers: []workflowsv1.ContainerNode{
				{
					Container: corev1.Container{
						Name:  "docker-build-diff-check-pr",
						Image: deployStepsDockerBuildDiffCheckImage,
						Args: []string{
							"./docker-build-diff-check-pr.sh",
							"{{inputs.parameters.repo}}",
							"/workspace",
							"{{inputs.parameters.revision}}",
							"{{inputs.parameters.revision-ref}}",
							"{{inputs.parameters.base-revision}}",
							"{{inputs.parameters.base-revision-ref}}",
							"{{inputs.parameters.dockerfile-path}}",
							"{{inputs.parameters.docker-context-dir}}",
							"/workspace/docker-build-pr-status.txt",
							"/repo",
						},
					},
				},
				{
					Container: corev1.Container{
						Name:  "main",
						Image: deployStepsDockerBuildImage,
						Args: []string{
							"pr",
							"--clone-path",
							"/workspace",
							"--dockerfile",
							"{{inputs.parameters.dockerfile-path}}",
							"--docker-context-dir",
							"{{inputs.parameters.docker-context-dir}}",
							"--status-file",
							"/workspace/docker-build-pr-status.txt",
						},
					},
					Dependencies: []string{"docker-build-diff-check-pr"},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				// Mount the shared repo workspace between the docker-build steps
				{
					Name:      "docker-build-pr-workspace",
					MountPath: "/workspace",
				},
			},
		},
		Volumes: []corev1.Volume{
			// Create a volume to share a repo workspace between the docker-build steps
			{
				Name: "docker-build-pr-workspace",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		// export the docker-build-pr-status to distinguish a "Skipped" docker build
		// which can cause further steps to be skipped
		// it can be accessed under the object: {tasks.<task-name>.outputs.parameters.docker-build-pr-status}}
		Outputs: workflowsv1.Outputs{
			Parameters: []workflowsv1.Parameter{
				{
					Name: "docker-build-pr-status",
					ValueFrom: &workflowsv1.ValueFrom{
						Path: "/workspace/docker-build-pr-status.txt",
					},
				},
			},
		},
	}

	// docker-build-test-publish
	// Useful for commit triggers. Will run and publish the image build
	DockerBuildTestPublishTemplate = workflowsv1.Template{
		Name: "docker-build-test-publish",
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				//  repo - the repo that will be cloned
				{
					Name: "repo",
				},
				// revision - the revision or commit sha that will be checked out
				{
					Name: "revision",
				},
				// revision-ref - the revision ref that will be used locally
				{
					Name: "revision-ref",
				},
				// dockerfile-path - the path to the Dockerfile which will be built
				{
					Name: "dockerfile-path",
				},
				// docker-context-dir - the path to the docker context dir that will be
				//                      used for the build
				{
					Name: "docker-context-dir",
				},
				// image-repo - the image repo prefix used to publish the image to
				{
					Name: "image-repo",
				},
				// dockerfile-dir - the image repo suffix used to publish the image to
				{
					Name: "dockerfile-dir",
				},
				// image-registry - the image registry used to publish the image to
				{
					Name:  "image-registry",
					Value: workflowsv1.AnyStringPtr(imageRegistry),
				},
			},
		},
		ContainerSet: &workflowsv1.ContainerSetTemplate{
			Containers: []workflowsv1.ContainerNode{
				{
					Container: corev1.Container{
						Name:  "docker-build-diff-check-commit",
						Image: deployStepsDockerBuildDiffCheckImage,
						Args: []string{
							"./docker-build-diff-check-commit.sh",
							"{{inputs.parameters.repo}}",
							"/workspace",
							"{{inputs.parameters.revision}}",
							"{{inputs.parameters.revision-ref}}",
							"{{inputs.parameters.dockerfile-path}}",
							"{{inputs.parameters.docker-context-dir}}",
							"/workspace/docker-build-commit-status.txt",
							"/repo",
						},
					},
				},
				{
					Container: corev1.Container{
						Name:  "main",
						Image: deployStepsDockerBuildImage,
						Args: []string{
							"commit",
							"--clone-path",
							"/workspace",
							"--revision-hash",
							"{{inputs.parameters.revision}}",
							"--revision-ref",
							"{{inputs.parameters.revision-ref}}",
							"--dockerfile",
							"{{inputs.parameters.dockerfile-path}}",
							"--docker-context-dir",
							"{{inputs.parameters.docker-context-dir}}",
							"--image-registry",
							"{{inputs.parameters.image-registry}}",
							"--image-repo",
							"{{inputs.parameters.image-repo}}",
							"--dockerfile-dir",
							"{{inputs.parameters.dockerfile-dir}}",
							"--status-file",
							"/workspace/docker-build-commit-status.txt",
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								// Mount the configuration so we can push the docker image
								Name:      "docker-config",
								MountPath: "/kaniko/.docker",
							},
						},
					},
					Dependencies: []string{"docker-build-diff-check-commit"},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				// Mount the shared repo workspace between the docker-build steps
				{
					Name:      "docker-build-commit-workspace",
					MountPath: "/workspace",
				},
			},
		},
		Volumes: []corev1.Volume{
			// Create a volume to share a repo workspace between the docker-build steps
			{
				Name: "docker-build-commit-workspace",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			// Mount the configuration so we can push the docker image
			{
				Name: "docker-config",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "regcred",
					},
				},
			},
		},
		// export the docker-build-commit-status to distinguish a "Skipped" docker build
		// which can cause further steps to be skipped
		// it can be accessed under the object: {tasks.<task-name>.outputs.parameters.docker-build-commit-status}}
		Outputs: workflowsv1.Outputs{
			Parameters: []workflowsv1.Parameter{
				{
					Name: "docker-build-commit-status",
					ValueFrom: &workflowsv1.ValueFrom{
						Path: "/workspace/docker-build-commit-status.txt",
					},
				},
			},
		},
	}

	// deploy-step-argocd
	// Deploy an image revision to the resource in the specified
	// argo cd config repo
	ArgoCDTemplate = workflowsv1.Template{
		Name: "deploy-step-argocd",
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				// deploy-repo - the repo containing the argo configs
				{
					Name: "deploy-repo",
				},
				// deploy-revision - the deploy-repo revision to update
				{
					Name: "deploy-revision",
				},
				// github-app-id - app id used for updating the deploy-repo
				{
					Name:  "github-app-id",
					Value: workflowsv1.AnyStringPtr(githubAppId),
				},
				// github-app-user-id - app id used for updating the deploy-repo
				{
					Name:  "github-app-user-id",
					Value: workflowsv1.AnyStringPtr(githubAppUserId),
				},
				// github-app-user-name - app user name for updating the deploy repo
				{
					Name:  "github-app-user-name",
					Value: workflowsv1.AnyStringPtr(githubAppUserName),
				},
				// resource-path - the path containing the resources to update
				{
					Name: "resource-path",
				},
				// image-repo - the target image repo prefix which will be updated in the argo configs
				{
					Name: "image-repo",
				},
				// build-revision - the target build revision used to update the argo configs
				{
					Name: "build-revision",
				},
				// dockerfile-dir - the target image repo suffix which will be updated in the argo configs
				{
					Name: "dockerfile-dir",
				},
				// image-registry - the image registry used for updating images
				{
					Name:  "image-registry",
					Value: workflowsv1.AnyStringPtr(imageRegistry),
				},
				// argocd-app-namespace - the ArgoCD application namespace
				{
					Name: "argocd-app-namespace",
				},
				// argocd-app-name - the ArgoCD application name
				{
					Name: "argocd-app-name",
				},
			},
		},
		Container: &corev1.Container{
			Image: deployStepsArgoCDImage,
			Args: []string{
				"./deploy-step-argocd.sh",
				"{{inputs.parameters.deploy-repo}}",
				"{{inputs.parameters.deploy-revision}}",
				"{{inputs.parameters.github-app-id}}",
				"{{inputs.parameters.github-app-user-id}}",
				"{{inputs.parameters.github-app-user-name}}",
				"/github-key/private-key.pem",
				"{{inputs.parameters.resource-path}}",
				"{{inputs.parameters.image-registry}}",
				"{{inputs.parameters.image-repo}}",
				"{{inputs.parameters.build-revision}}",
				"{{inputs.parameters.dockerfile-dir}}",
				"{{inputs.parameters.argocd-app-namespace}}",
				"{{inputs.parameters.argocd-app-name}}",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					// Mount the configuration so we can push to github
					Name:      "github-key",
					MountPath: "/github-key",
				},
			},
		},
		Volumes: []corev1.Volume{
			// Mount the configuration so we can push to github
			{
				Name: "github-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "github-key",
					},
				},
			},
		},
		Synchronization: &workflowsv1.Synchronization{
			Mutexes: []*workflowsv1.Mutex{
				{
					// todo may need to include namespace
					Name: "{{inputs.parameters.deploy-repo}}/{{inputs.parameters.resource-path}}",
				},
			},
		},
	}

	// deploy-step-github-check-complete
	// Completes a GitHub check for the specified commit.
	GitHubCheckCompleteTemplate = workflowsv1.Template{
		Name:                  "deploy-step-github-check-complete",
		ActiveDeadlineSeconds: &activeDeadlineSeconds5m, // 5m
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				// github-app-id - app id used for updating GitHub checks
				{
					Name:  "github-app-id",
					Value: workflowsv1.AnyStringPtr(githubAppId),
				},
				// repo-short - repo in the format <org-name>/<repo-name>
				{
					Name: "repo-short",
				},
				// workflow-url - the url to the Argo Workflow UI of this workflow run
				{
					Name:  "workflow-url",
					Value: workflowsv1.AnyStringPtr("https://argo.osoriano.com/workflows/{{workflow.namespace}}/{{workflow.name}}"),
				},
				// event-type - a short description of the event type (e.g. PR or Commit)
				{
					Name: "event-type",
				},
				// check-run-id - the id of the GitHub Status Check to complete
				{
					Name: "check-run-id",
				},
				// workflow-status - status that will be used to complete the GitHub Check
				{
					Name: "workflow-status",
				},
			},
		},
		Container: &corev1.Container{
			Image: deployStepsGitHubCheckImage,
			Args: []string{
				"./github-check-complete.sh",
				"{{inputs.parameters.github-app-id}}",
				"/github-key/private-key.pem",
				"{{inputs.parameters.repo-short}}",
				"{{inputs.parameters.workflow-url}}",
				"{{inputs.parameters.event-type}}",
				"{{inputs.parameters.check-run-id}}",
				"{{inputs.parameters.workflow-status}}",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					// Mount the configuration so we can update the git status
					Name:      "github-key",
					MountPath: "/github-key",
				},
			},
		},
		Volumes: []corev1.Volume{
			// Mount the configuration so we can update the git status
			{
				Name: "github-key",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "github-key",
					},
				},
			},
		},
	}

	CICDTemplate = workflowsv1.ClusterWorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflows.ClusterWorkflowTemplateKind,
			APIVersion: workflows.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cicd-templates",
		},
		Spec: workflowsv1.WorkflowSpec{
			Templates: []workflowsv1.Template{
				GitHubCheckStartTemplate,
				DockerBuildTestTemplate,
				DockerBuildTestPublishTemplate,
				ArgoCDTemplate,
				GitHubCheckCompleteTemplate,
			},
		},
	}
)

func SyncWorkflowTemplates(client ctrlclient.Client) {
	for {
		log.Info("syncing cluster workflow templates")
		cicdTemplate := &workflowsv1.ClusterWorkflowTemplate{
			TypeMeta:   CICDTemplate.TypeMeta,
			ObjectMeta: CICDTemplate.ObjectMeta,
		}

		op, err := ctrl.CreateOrUpdate(
			context.Background(),
			client,
			cicdTemplate,
			func() error {
				cicdTemplate.Spec = CICDTemplate.Spec
				return nil
			},
		)
		if err != nil {
			log.Error(err, "unable to sync cluster workflow templates. Retrying...")
			time.Sleep(5 * time.Second)
			continue
		}
		log.Info("synced cluster workflow templates", "operation", op)
		break
	}
}
