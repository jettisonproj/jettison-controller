package v1alpha1

const (
	// The available trigger sources. Should be pascalcased
	gitHubPullRequestTriggerSource = "GitHubPullRequest"
	gitHubPushTriggerSource        = "GitHubPush"
)

var (
	triggerSources = []string{
		gitHubPullRequestTriggerSource,
		gitHubPushTriggerSource,
	}
)

// The common json fields for triggers
// Satisfies the BaseTrigger interface
type BaseTriggerFields struct {
	// The name of trigger for the flow. This can be a description
	TriggerName string `json:"triggerName"`
	// The type of trigger for the flow
	TriggerSource string `json:"triggerSource"`
}

func (t BaseTriggerFields) GetTriggerName() string {
	return t.TriggerName
}

func (t BaseTriggerFields) GetTriggerSource() string {
	return t.TriggerSource
}

type GitHubPullRequestTrigger struct {
	BaseTriggerFields

	// The url of the GitHub repo. For example: https://github.com/jettisonproj/rollouts-demo.git
	RepoUrl string `json:"repoUrl"`
	// Optional base ref or branch that the PR will be merged to. This is typically the default
	// branch name such as "main" or "master"
	// Defaults to "main"
	// +optional
	BaseRef *string `json:"baseRef,omitempty"`
	// Optional list of GitHub pull request event types that will trigger the PR workflow.
	// See https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request
	// Defaults to [opened, reopened, synchronize]
	// +optional
	PullRequestEvents []string `json:"pullRequestEvents,omitempty"`
}

type GitHubPushTrigger struct {
	BaseTriggerFields

	// The url of the GitHub repo. For example: https://github.com/jettisonproj/rollouts-demo.git
	RepoUrl string `json:"repoUrl"`
	// Optional base ref for the push event. This is typically the default
	// branch name such as "main" or "master"
	// Defaults to "main"
	// +optional
	BaseRef *string `json:"baseRef,omitempty"`
}
