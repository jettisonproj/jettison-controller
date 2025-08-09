package gitutil

import (
	"fmt"
	"strings"
)

const (
	gitSuffix = ".git"
)

func GetRepoOrgName(repoUrl string) (string, string, error) {
	repoUrl = strings.TrimSuffix(repoUrl, gitSuffix)
	repoUrlParts := strings.Split(repoUrl, "/")
	if len(repoUrlParts) < 2 {
		return "", "", fmt.Errorf("invalid repoUrl: %s", repoUrl)
	}
	repoOrg := repoUrlParts[len(repoUrlParts)-2]
	repoName := repoUrlParts[len(repoUrlParts)-1]
	return repoOrg, repoName, nil
}
