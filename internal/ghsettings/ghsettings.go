package ghsettings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v74/github"
	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log               = ctrl.Log.WithName("ghsettings")
	syncRetryInterval = 5 * time.Second
)

func SyncGitHubSettings(flowClient ctrlclient.Client) {
	ghClient := github.NewClient(nil) // Use nil for new http client
	for {
		log.Info("syncing GitHub settings")

		flows := &v1alpha1.FlowList{}
		err := flowClient.List(context.Background(), flows)
		if err != nil {
			log.Error(err, "failed to get flows for ghsettings. Retrying...")
			time.Sleep(syncRetryInterval)
			continue
		}
		err = syncGitHubSettingsForFlows(ghClient, flows)
		if err != nil {
			log.Error(err, "failed to sync ghsettings for flows. Retrying...")
			time.Sleep(syncRetryInterval)
			continue
		}
		log.Info("synced GitHub settings")
		break
	}
}

func syncGitHubSettingsForFlows(ghClient *github.Client, flows *v1alpha1.FlowList) error {
	// First collect repo urls from flows
	repoUrls := make(map[string]bool)
	for _, flow := range flows.Items {
		repoUrl, err := getRepoUrlForFlow(flow)
		if err != nil {
			return err
		}
		repoUrls[repoUrl] = true
	}

	// Then, sync settings for each repo
	for repoUrl := range repoUrls {
		err := syncGitHubSettingsForRepo(ghClient, repoUrl)
		if err != nil {
			return err
		}
	}
	return nil
}

func getRepoUrlForFlow(flow v1alpha1.Flow) (string, error) {
	flowTriggers, _, err := flow.ProcessFlow()
	if err != nil {
		return "", err
	}

	for _, flowTrigger := range flowTriggers {
		switch trigger := flowTrigger.(type) {
		case *v1alpha1.GitHubPullRequestTrigger:
			return trigger.RepoUrl, nil
		case *v1alpha1.GitHubPushTrigger:
			return trigger.RepoUrl, nil
		default:
			return "", fmt.Errorf("unknown trigger type for flow %s: %T", flow.Name, trigger)
		}
	}
	return "", fmt.Errorf("did not find trigger for flow: %s", flow.Name)
}

// todo implement me
func syncGitHubSettingsForRepo(ghClient *github.Client, repoUrl string) error {
	log.Info("syncing GitHub settings", "repoUrl", repoUrl)
	orgs, _, err := ghClient.Organizations.List(context.Background(), "osoriano", nil)
	if err != nil {
		return err
	}
	for _, org := range orgs {
		log.Info("found org", "name", org.Name)
	}
	return nil
}
