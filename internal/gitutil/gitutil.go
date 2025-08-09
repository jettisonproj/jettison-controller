package gitutil

import (
	"fmt"
	"strings"
)

const (
	gitPrefix = "https://github.com/"
	gitSuffix = ".git"
)

func GetRepoOrgName(repoUrl string) (string, string, error) {

	// Accept repoUrls with and without the git suffix
	repoUrl = strings.TrimSuffix(repoUrl, gitSuffix)

	// Currently, only GitHub urls are supported
	repoPath, found := strings.CutPrefix(repoUrl, gitPrefix)
	if !found {
		return "", "", fmt.Errorf("invalid repoUrl prefix: %s", repoUrl)
	}

	repoUrlParts := strings.Split(repoPath, "/")
	if len(repoUrlParts) != 2 {
		return "", "", fmt.Errorf("invalid repoUrl path: %s", repoUrl)
	}
	repoOrg := repoUrlParts[0]
	repoName := repoUrlParts[1]
	return repoOrg, repoName, nil
}

func GetSourceRepoGlob(repoOrg string) string {
	return fmt.Sprintf(
		"%s%s/*",
		gitPrefix,
		repoOrg,
	)
}
