package gitutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jettisonproj/jettison-controller/internal/gitutil"
)

func TestGetRepoOrgName(t *testing.T) {
	testCases := []struct {
		name             string
		repoUrl          string
		expectedRepoOrg  string
		expectedRepoName string
	}{
		{
			"TestGetRepoOrgNameWithSuffix",
			"https://github.com/jettisonproj/jettison-controller.git",
			"jettisonproj",
			"jettison-controller",
		},
		{
			"TestGetRepoOrgNameWithoutSuffix",
			"https://github.com/jettisonproj/jettison-controller",
			"jettisonproj",
			"jettison-controller",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoOrg, repoName, err := gitutil.GetRepoOrgName(tc.repoUrl)
			require.Nilf(t, err, "error getting repo org name")
			require.Equalf(t, tc.expectedRepoOrg, repoOrg, "unexpected repo org")
			require.Equalf(t, tc.expectedRepoName, repoName, "unexpected repo name")
		})
	}
}

func TestGetSourceRepoGlob(t *testing.T) {
	sourceRepoGlob := gitutil.GetSourceRepoGlob("jettisonproj")
	require.Equalf(
		t,
		"https://github.com/jettisonproj/*",
		sourceRepoGlob,
		"unexpected source repo glob",
	)
}
