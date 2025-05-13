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
	// GitHub App ID of the argo config updater
	githubAppId = "955485"
	// GitHub App User ID of the argo config updater
	githubAppUserId = "176784108"
	// GitHub App User Name of the argo config updater
	githubAppUserName = "argocd-config-updater[bot]"
	// Deploy Step Image for GitHub Checks
	deployStepsGitHubCheckImage = "osoriano/deploy-steps-github-check:sha-9c772691d7978630c9981ef2683194a966d4a606"
	// Deploy Step Image for ArgoCD Config Update
	deployStepsArgoCDImage = "osoriano/deploy-steps-argocd:sha-9c772691d7978630c9981ef2683194a966d4a606"
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
	// Useful for PR build. Will run the image build, but skips publishing it
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
			},
			Artifacts: []workflowsv1.Artifact{
				{
					Name: "repo-source",
					Path: "/repo",
					ArtifactLocation: workflowsv1.ArtifactLocation{
						Git: &workflowsv1.GitArtifact{
							Repo:     "{{inputs.parameters.repo}}",
							Revision: "{{inputs.parameters.revision}}",
						},
					},
				},
			},
		},
		Container: &corev1.Container{
			Image: "gcr.io/kaniko-project/executor:latest",
			Args: []string{
				"--dockerfile=/repo/{{inputs.parameters.dockerfile-path}}",
				"--context=dir:///repo/{{inputs.parameters.docker-context-dir}}",
				"--no-push",
			},
		},
	}

	// docker-build-test-publish
	// Useful for commit build. Will run and publish the image build
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
				// dockerfile-path - the path to the Dockerfile which will be built
				{
					Name: "dockerfile-path",
				},
				// docker-context-dir - the path to the docker context dir that will be
				//                      used for the build
				{
					Name: "docker-context-dir",
				},
				// image-repo - the image repo used to publish the image to
				{
					Name: "image-repo",
				},
				// image-registry - the image registry used to publish the image to
				{
					Name:  "image-registry",
					Value: workflowsv1.AnyStringPtr(imageRegistry),
				},
			},
			Artifacts: []workflowsv1.Artifact{
				{
					Name: "repo-source",
					Path: "/repo",
					ArtifactLocation: workflowsv1.ArtifactLocation{
						Git: &workflowsv1.GitArtifact{
							Repo:     "{{inputs.parameters.repo}}",
							Revision: "{{inputs.parameters.revision}}",
						},
					},
				},
			},
		},
		Container: &corev1.Container{
			Image: "gcr.io/kaniko-project/executor:latest",
			Args: []string{
				"--dockerfile=/repo/{{inputs.parameters.dockerfile-path}}",
				"--context=dir:///repo/{{inputs.parameters.docker-context-dir}}",
				"--destination={{=inputs.parameters['image-registry'] + inputs.parameters['image-repo'] + join(split(inputs.parameters['dockerfile-path'], '/')[:-1], ) + ':' + inputs.parameters.revision}}",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					// Mount the configuration so we can push the docker image
					Name:      "docker-config",
					MountPath: "/kaniko/.docker",
				},
			},
		},
		Volumes: []corev1.Volume{
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
				// image-repo - the target image repo which will be updated in the argo configs
				{
					Name: "image-repo",
				},
				// build-revision - the target build revision used to update the argo configs
				{
					Name: "build-revision",
				},
				// deployment-approved - Optionally specify the approval status of the deployment.
				//                       Defaults to 'YES'. If set to 'NO', fails the deployment
				{
					Name:    "deployment-approved",
					Default: workflowsv1.AnyStringPtr("YES"),
				},
				// image-registry - the image registry used for updating images
				{
					Name:  "image-registry",
					Value: workflowsv1.AnyStringPtr(imageRegistry),
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
				"{{inputs.parameters.deployment-approved}}",
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

	// approval
	// Waits until this step to be approved before continuing
	ApprovalTemplate = workflowsv1.Template{
		Name:                  "approval",
		ActiveDeadlineSeconds: &activeDeadlineSeconds3d,
		Suspend:               &workflowsv1.SuspendTemplate{},
		Inputs: workflowsv1.Inputs{
			Parameters: []workflowsv1.Parameter{
				{
					Name:        "approve",
					Description: workflowsv1.AnyStringPtr("Choose YES to continue workflow and deploy to prod"),
					Default:     workflowsv1.AnyStringPtr("NO"),
					Enum: []workflowsv1.AnyString{
						workflowsv1.AnyString("YES"),
						workflowsv1.AnyString("NO"),
					},
				},
			},
		},
		// export a global parameter so that the approval is  available in future
		// steps. It can be accessed under the object: workflow.outputs.parameters
		Outputs: workflowsv1.Outputs{
			Parameters: []workflowsv1.Parameter{
				{
					Name: "approve",
					ValueFrom: &workflowsv1.ValueFrom{
						Supplied: &workflowsv1.SuppliedValueFrom{},
					},
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
				ApprovalTemplate,
				GitHubCheckCompleteTemplate,
			},
		},
	}
)

func CreateWorkflowTemplates(client ctrlclient.Client) {
	for {
		log.Info("creating cluster workflow templates")
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
			log.Error(err, "unable to create cluster workflow templates. Retrying...")
			time.Sleep(5 * time.Second)
			continue
		}
		log.Info("created cluster workflow templates", "operation", op)
		break
	}
}
