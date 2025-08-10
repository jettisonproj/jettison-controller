package eventsourcebuilder

import (
	eventsource "github.com/argoproj/argo-events/pkg/apis/eventsource"
	eventsv1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	eventSourceNamespace   = "argo-events"
	eventSourceName        = "github"
	EventSourceSectionName = "ghwebhook"
)

var (
	log = ctrl.Log.WithName("eventsourcebuilder")

	githubEventSource = eventsv1.GithubEventSource{
		// GitHub will send events to following port and endpoint
		Webhook: &eventsv1.WebhookContext{
			// Endpoint to listen to events on
			Endpoint: "/push",
			// HTTP request method to allow. In this case, only POST requests are accepted
			Method: "POST",
			// Port to run internal HTTP server on
			Port: "12000",
			// URL the event-source will use to register at Github.
			// This url must be reachable from outside the cluster.
			// The name for the service is in `<event-source-name>-eventsource-svc` format.
			// You will need to create an Ingress or Openshift Route for the event-source service so that it can be reached from GitHub.
			URL: "https://argo-github.osoriano.com",
		},
		// Type of events to listen to.
		// Following listens to everything, hence *
		// You can find more info on https://developer.github.com/v3/activity/events/types/
		Events: []string{"*"},
		// APIToken refers to K8s secret that stores the github api token
		// if apiToken is provided controller will create webhook on GitHub repo
		// +optional
		APIToken: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				// Name of the K8s secret that contains the access token
				Name: "github-access",
			},
			// Key within the K8s secret whose corresponding value (must be base64 encoded) is access token
			Key: "token",
		},
		// WebhookSecret refers to K8s secret that stores the github hook secret
		// +optional
		WebhookSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				// Name of the K8s secret that contains the hook secret
				Name: "github-access",
			},
			// Key within the K8s secret whose corresponding value (must be base64 encoded) is hook secret
			Key: "secret",
		},
		// Type of the connection between event-source and Github.
		// You should set it to false to avoid man-in-the-middle and other attacks.
		Insecure: false,
		// Determines if notifications are sent when the webhook is triggered
		Active: true,
		// The media type used to serialize the payloads
		ContentType: "json",
		// Reposities that will be configured with webhooks
		// This is dynamically generated based on the existing flow repo urls
		Repositories: nil,
	}

	eventSource = eventsv1.EventSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       eventsource.Kind,
			APIVersion: eventsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: eventSourceNamespace,
			Name:      eventSourceName,
		},
		Spec: eventsv1.EventSourceSpec{
			Service: &eventsv1.Service{
				Ports: []corev1.ServicePort{
					{
						Name:       EventSourceSectionName,
						Port:       12000,
						TargetPort: intstr.FromInt(12000),
					},
				},
			},
			Github: map[string]eventsv1.GithubEventSource{
				EventSourceSectionName: githubEventSource,
			},
		},
	}
)

func BuildEventSource(repoOrg string, repoName string) *eventsv1.EventSource {
	return &eventsv1.EventSource{
		TypeMeta:   eventSource.TypeMeta,
		ObjectMeta: eventSource.ObjectMeta,
		Spec:       getEventSourceSpecWithRepos(repoOrg, repoName),
	}
}

func getEventSourceSpecWithRepos(repoOrg string, repoName string) eventsv1.EventSourceSpec {
	githubEventSourceWithRepos := githubEventSource
	githubEventSourceWithRepos.Repositories = []eventsv1.OwnedRepositories{
		{
			Owner: repoOrg,
			Names: []string{repoName},
		},
	}

	eventSourceSpecWithRepos := eventSource.Spec
	eventSourceSpecWithRepos.Github = map[string]eventsv1.GithubEventSource{
		EventSourceSectionName: githubEventSourceWithRepos,
	}

	return eventSourceSpecWithRepos
}
