package ghsettings

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	githubKey                = "jettisonproj.private-key.pem"
	appId              int64 = 1682308
	defaultRulesetName       = "default"
	defaultBranch            = "~DEFAULT_BRANCH"
)

var (
	log               = ctrl.Log.WithName("ghsettings")
	syncRetryInterval = 5 * time.Second

	rulesetSourceType   = github.RulesetSourceTypeRepository
	rulesetTargetBranch = github.RulesetTargetBranch
	falseFlag           = false
	trueFlag            = true
	appIdFlag           = appId

	jettisonRequiredCheck = &github.RuleStatusCheck{
		// The Context should be kept in sync with:
		// deploy-steps/github-check
		Context:       "Jettison PR Flow",
		IntegrationID: &appIdFlag,
	}

	jettisonRequiredChecks = []*github.RuleStatusCheck{
		jettisonRequiredCheck,
	}

	jettisonStatusChecks = &github.RequiredStatusChecksRuleParameters{
		DoNotEnforceOnCreate:             &falseFlag,
		RequiredStatusChecks:             jettisonRequiredChecks,
		StrictRequiredStatusChecksPolicy: true,
	}

	jettisonAllowedMergeMethods = []github.PullRequestMergeMethod{
		github.PullRequestMergeMethodSquash,
	}

	jettisonPrRules = &github.PullRequestRuleParameters{
		AllowedMergeMethods:            jettisonAllowedMergeMethods,
		DismissStaleReviewsOnPush:      true,
		RequireCodeOwnerReview:         false,
		RequireLastPushApproval:        false,
		RequiredApprovingReviewCount:   0,
		RequiredReviewThreadResolution: true,
	}

	jettisonRulesetRules = &github.RepositoryRulesetRules{
		RequiredLinearHistory: &github.EmptyRuleParameters{},
		Deletion:              &github.EmptyRuleParameters{},
		PullRequest:           jettisonPrRules,
		RequiredStatusChecks:  jettisonStatusChecks,
		NonFastForward:        &github.EmptyRuleParameters{},
	}

	jettisonRulesetConditions = &github.RepositoryRulesetConditions{
		RefName: &github.RepositoryRulesetRefConditionParameters{
			Include: []string{defaultBranch},
			Exclude: []string{},
		},
	}

	jettisonRuleset = github.RepositoryRuleset{
		Name:        defaultRulesetName,
		Target:      &rulesetTargetBranch,
		SourceType:  &rulesetSourceType,
		Enforcement: github.RulesetEnforcementActive,
		Conditions:  jettisonRulesetConditions,
		Rules:       jettisonRulesetRules,
	}
)

// Configure the GitHub client with app auth
func GetGitHubClient() (*ghinstallation.AppsTransport, *github.Client, error) {
	ghTransport, err := ghinstallation.NewAppsTransportKeyFromFile(
		http.DefaultTransport,
		appId,
		githubKey,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get GitHub client transport: %s", err)
	}

	ghClient := github.NewClient(&http.Client{Transport: ghTransport})
	return ghTransport, ghClient, nil
}

func SyncGitHubSettingsForRepo(
	ctx context.Context,
	appsTransport *ghinstallation.AppsTransport,
	ghClient *github.Client,
	repoOrg string,
	repoName string,
) error {
	log.Info("syncing GitHub settings", "repoOrg", repoOrg, "repoName", repoName)
	installation, _, err := ghClient.Apps.FindRepositoryInstallation(ctx, repoOrg, repoName)
	if err != nil {
		return fmt.Errorf("failed to get installation id for repo: %s/%s", repoOrg, repoName)
	}
	log.Info("found installation", "id", installation.ID, "repoOrg", repoOrg, "repoName", repoName)

	ghRepoTransport := ghinstallation.NewFromAppsTransport(appsTransport, *installation.ID)
	ghRepoClient := github.NewClient(&http.Client{Transport: ghRepoTransport})

	repo, _, err := ghRepoClient.Repositories.Get(ctx, repoOrg, repoName)
	if err != nil {
		return fmt.Errorf("failed to fetch repo %s/%s: %s", repoOrg, repoName, err)
	}

	// Update repo settings if needed

	hasDiff := false

	if repo.AllowMergeCommit == nil {
		log.Info(
			"setting allow merge commit to false to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowMergeCommit = &falseFlag
		hasDiff = true
	} else if *repo.AllowMergeCommit {
		log.Info(
			"setting allow merge commit to false",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowMergeCommit = &falseFlag
		hasDiff = true
	}

	if repo.AllowSquashMerge == nil {
		log.Info(
			"setting allow squash merge to true to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowSquashMerge = &trueFlag
		hasDiff = true
	} else if !*repo.AllowSquashMerge {
		log.Info(
			"setting allow squash merge to true",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowSquashMerge = &trueFlag
		hasDiff = true
	}

	if repo.AllowRebaseMerge == nil {
		log.Info(
			"setting allow rebase merge to false to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowRebaseMerge = &falseFlag
		hasDiff = true
	} else if *repo.AllowRebaseMerge {
		log.Info(
			"setting allow rebase merge to false",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowRebaseMerge = &falseFlag
		hasDiff = true
	}

	if repo.AllowUpdateBranch == nil {
		log.Info(
			"setting allow update branch to true to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowUpdateBranch = &trueFlag
		hasDiff = true
	} else if !*repo.AllowUpdateBranch {
		log.Info(
			"setting allow update branch to true",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowUpdateBranch = &trueFlag
		hasDiff = true
	}

	if repo.AllowAutoMerge == nil {
		log.Info(
			"setting allow auto merge to true to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowAutoMerge = &trueFlag
		hasDiff = true
	} else if !*repo.AllowAutoMerge {
		log.Info(
			"setting allow auto merge to true",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.AllowAutoMerge = &trueFlag
		hasDiff = true
	}

	if repo.DeleteBranchOnMerge == nil {
		log.Info(
			"setting delete branch on merge to true to override default",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.DeleteBranchOnMerge = &trueFlag
		hasDiff = true
	} else if !*repo.DeleteBranchOnMerge {
		log.Info(
			"setting delete branch on merge to true",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		repo.DeleteBranchOnMerge = &trueFlag
		hasDiff = true
	}

	if hasDiff {
		_, _, err = ghRepoClient.Repositories.Edit(ctx, repoOrg, repoName, repo)
		if err != nil {
			return fmt.Errorf("failed to edit repo settings %s/%s: %s", repoOrg, repoName, err)
		}
	} else {
		log.Info(
			"repo settings already in sync",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
	}

	// Update rulesets if needed
	rulesets, _, err := ghRepoClient.Repositories.GetAllRulesets(ctx, repoOrg, repoName, nil)
	if err != nil {
		return fmt.Errorf("failed to get rulesets for %s/%s: %s", repoOrg, repoName, err)
	}
	var defaultRuleset *github.RepositoryRuleset
	for _, ruleset := range rulesets {
		if ruleset.Name == defaultRulesetName {
			// The ruleset list api does not return full details. Call the get api here
			defaultRuleset, _, err = ghRepoClient.Repositories.GetRuleset(ctx, repoOrg, repoName, *ruleset.ID, false)
			if err != nil {
				return fmt.Errorf("failed to get ruleset for %s/%s: %s", repoOrg, repoName, err)
			}
			break
		}
	}

	if defaultRuleset == nil {
		log.Info(
			"No existing default ruleset. Creating...",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)
		_, _, err := ghRepoClient.Repositories.CreateRuleset(ctx, repoOrg, repoName, jettisonRuleset)
		if err != nil {
			return fmt.Errorf("failed to create repo rulesets %s/%s: %s", repoOrg, repoName, err)
		}
	} else {
		log.Info(
			"found existing default ruleset",
			"repoOrg", repoOrg,
			"repoName", repoName,
		)

		hasDiff := false

		if defaultRuleset.Target == nil {
			log.Info(
				"setting ruleset target to override default",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			defaultRuleset.Target = &rulesetTargetBranch
			hasDiff = true
		} else if *defaultRuleset.Target != rulesetTargetBranch {
			log.Info(
				"setting ruleset target",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			defaultRuleset.Target = &rulesetTargetBranch
			hasDiff = true
		}

		if defaultRuleset.SourceType == nil {
			log.Info(
				"setting ruleset source type to override default",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			defaultRuleset.SourceType = &rulesetSourceType
			hasDiff = true
		} else if *defaultRuleset.SourceType != rulesetSourceType {
			log.Info(
				"setting ruleset source type",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			defaultRuleset.SourceType = &rulesetSourceType
			hasDiff = true
		}

		if defaultRuleset.Enforcement != github.RulesetEnforcementActive {
			log.Info(
				"setting ruleset enforcement",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			defaultRuleset.Enforcement = github.RulesetEnforcementActive
			hasDiff = true
		}

		conditions := defaultRuleset.Conditions
		if conditions == nil {
			log.Info(
				"setting ruleset condition to override default",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			conditions = jettisonRulesetConditions
			defaultRuleset.Conditions = conditions
			hasDiff = true
		} else {
			refName := conditions.RefName
			if refName == nil {
				log.Info(
					"setting ruleset condition refName to override default",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				refName = jettisonRulesetConditions.RefName
				conditions.RefName = refName
				hasDiff = true
			} else {
				if len(refName.Include) != 1 || (len(refName.Include) == 1 && refName.Include[0] != defaultBranch) {
					log.Info(
						"setting ruleset condition include",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					refName.Include = []string{defaultBranch}
					hasDiff = true
				}
				// exclude cannot be nil. Otherwise, the api rejects the call
				if refName.Exclude == nil || len(refName.Exclude) != 0 {
					log.Info(
						"setting ruleset condition exclude",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					refName.Exclude = []string{}
					hasDiff = true
				}
			}
		}

		rules := defaultRuleset.Rules
		if rules == nil {
			log.Info(
				"setting ruleset rules to override default",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
			rules := jettisonRulesetRules
			defaultRuleset.Rules = rules
			hasDiff = true
		} else {
			if rules.RequiredLinearHistory == nil {
				log.Info(
					"setting ruleset rules required linear history",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				rules.RequiredLinearHistory = &github.EmptyRuleParameters{}
				hasDiff = true
			}

			if rules.Deletion == nil {
				log.Info(
					"setting ruleset rules deletion",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				rules.Deletion = &github.EmptyRuleParameters{}
				hasDiff = true
			}

			pullRequest := rules.PullRequest
			if pullRequest == nil {
				log.Info(
					"setting ruleset pr rule to override default",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				pullRequest = jettisonPrRules
				rules.PullRequest = pullRequest
				hasDiff = true
			} else {
				if len(pullRequest.AllowedMergeMethods) != 1 || pullRequest.AllowedMergeMethods[0] != github.PullRequestMergeMethodSquash {
					log.Info(
						"setting ruleset pr rule merge methods",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.AllowedMergeMethods = jettisonAllowedMergeMethods
					hasDiff = true
				}

				if !pullRequest.DismissStaleReviewsOnPush {
					log.Info(
						"setting ruleset pr rule dismiss stale reviews",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.DismissStaleReviewsOnPush = true
					hasDiff = true
				}

				if pullRequest.RequireCodeOwnerReview {
					log.Info(
						"setting ruleset pr rule require code owner review",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.RequireCodeOwnerReview = false
					hasDiff = true
				}

				if pullRequest.RequireLastPushApproval {
					log.Info(
						"setting ruleset pr rule require last push approval",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.RequireLastPushApproval = false
					hasDiff = true
				}

				if pullRequest.RequiredApprovingReviewCount != 0 {
					log.Info(
						"setting ruleset pr rule required approvals",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.RequiredApprovingReviewCount = 0
					hasDiff = true
				}

				if !pullRequest.RequiredReviewThreadResolution {
					log.Info(
						"setting ruleset pr rule review thread resolution",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					pullRequest.RequiredReviewThreadResolution = true
					hasDiff = true
				}
			}

			statusChecks := rules.RequiredStatusChecks
			if statusChecks == nil {
				log.Info(
					"setting ruleset status checks to override defaults",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				statusChecks = jettisonStatusChecks
				rules.RequiredStatusChecks = statusChecks
				hasDiff = true
			} else {
				if statusChecks.DoNotEnforceOnCreate == nil {
					log.Info(
						"setting ruleset status checks do not enforce on create to override defaults",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					statusChecks.DoNotEnforceOnCreate = &falseFlag
					hasDiff = true
				} else if *statusChecks.DoNotEnforceOnCreate {
					log.Info(
						"setting ruleset status checks do not enforce on create",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					statusChecks.DoNotEnforceOnCreate = &falseFlag
					hasDiff = true
				}

				requiredChecks := statusChecks.RequiredStatusChecks
				if len(requiredChecks) != 1 || requiredChecks[0].Context != jettisonRequiredCheck.Context || requiredChecks[0].IntegrationID == nil || *requiredChecks[0].IntegrationID != appId {
					log.Info(
						"setting ruleset required status checks",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					requiredChecks = jettisonRequiredChecks
					statusChecks.RequiredStatusChecks = requiredChecks
					hasDiff = true
				}

				if !statusChecks.StrictRequiredStatusChecksPolicy {
					log.Info(
						"setting ruleset status checks strict required",
						"repoOrg", repoOrg,
						"repoName", repoName,
					)
					statusChecks.StrictRequiredStatusChecksPolicy = true
					hasDiff = true
				}
			}

			if rules.NonFastForward == nil {
				log.Info(
					"setting ruleset rules non fast forward",
					"repoOrg", repoOrg,
					"repoName", repoName,
				)
				rules.NonFastForward = &github.EmptyRuleParameters{}
				hasDiff = true
			}
		}

		if hasDiff {
			_, _, err := ghRepoClient.Repositories.UpdateRuleset(ctx, repoOrg, repoName, *defaultRuleset.ID, *defaultRuleset)
			if err != nil {
				return fmt.Errorf("failed to edit repo rulesets %s/%s: %s", repoOrg, repoName, err)
			}
		} else {
			log.Info(
				"repo rulesets already in sync",
				"repoOrg", repoOrg,
				"repoName", repoName,
			)
		}
	}

	// For package settings, it is not available to view. See:
	// https://github.com/orgs/community/discussions/24636
	// Also, the edit API is missing here:
	// https://docs.github.com/en/rest/packages/packages?apiVersion=2022-11-28#list-packages-for-an-organization
	// containerPackageType := "container"
	// packageListOptions := &github.PackageListOptions{
	// 	PackageType: &containerPackageType,
	// }
	// packages, _, err := ghRepoClient.Organizations.ListPackages(ctx, repoOrg, packageListOptions)
	// log.Info("packages", "repoOrg", repoOrg, "packages", fmt.Sprintf("%#v", packages))

	return nil
}
