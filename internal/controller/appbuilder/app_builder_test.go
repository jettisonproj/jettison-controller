package appbuilder_test

import (
	"fmt"
	"testing"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/require"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	"github.com/jettisonproj/jettison-controller/internal/controller/appbuilder"
	"github.com/jettisonproj/jettison-controller/internal/testutil"
)

const (
	testdataDir = "../../../testdata"
)

func TestBuildArgoApps(t *testing.T) {
	prFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-pr-minimal.yaml")
	pushFlowPath := fmt.Sprintf("%s/%s", testdataDir, "github-push-minimal.yaml")
	prFlow, err := testutil.ParseYaml[v1alpha1.Flow](prFlowPath)
	require.Nilf(t, err, "failed to parse pr flow: %s", prFlowPath)
	_, prFlowSteps, err := prFlow.PreProcessFlow()
	require.Nilf(t, err, "failed to preprocess pr flow: %s", prFlowPath)
	pushFlow, err := testutil.ParseYaml[v1alpha1.Flow](pushFlowPath)
	require.Nilf(t, err, "failed to parse push flow: %s", pushFlowPath)
	_, pushFlowSteps, err := pushFlow.PreProcessFlow()
	require.Nilf(t, err, "failed to preprocess push flow: %s", prFlowPath)

	devAppPath := fmt.Sprintf("%s/%s", testdataDir, "application-dev.yaml")
	stagingAppPath := fmt.Sprintf("%s/%s", testdataDir, "application-staging.yaml")
	prodAppPath := fmt.Sprintf("%s/%s", testdataDir, "application-prod.yaml")
	devApp, err := testutil.ParseYaml[cdv1.Application](devAppPath)
	require.Nilf(t, err, "failed to parse dev application: %s", devAppPath)
	stagingApp, err := testutil.ParseYaml[cdv1.Application](stagingAppPath)
	require.Nilf(t, err, "failed to parse staging application: %s", stagingAppPath)
	prodApp, err := testutil.ParseYaml[cdv1.Application](prodAppPath)
	require.Nilf(t, err, "failed to parse prod application: %s", prodAppPath)

	appProjectPath := fmt.Sprintf("%s/%s", testdataDir, "app-project.yaml")
	appProject, err := testutil.ParseYaml[cdv1.AppProject](appProjectPath)
	require.Nilf(t, err, "failed to parse app project: %s", appProjectPath)

	prProjects, prApplications, err := appbuilder.BuildArgoApps(prFlowSteps)
	require.Nilf(t, err, "failed get applications for pr flow")
	require.Emptyf(t, prApplications, "expected empty applications for pr flow")
	require.Emptyf(t, prProjects, "expected empty app projects for pr flow")

	pushProjects, pushApplications, err := appbuilder.BuildArgoApps(pushFlowSteps)
	require.Nilf(t, err, "failed get applications for push flow")
	require.Lenf(t, pushApplications, 3, "expected dev, staging, and prod applications for push flow")
	require.Lenf(t, pushProjects, 1, "expected single app project for push flow")

	require.Equalf(t, devApp, pushApplications[0], "unexpected dev app")
	require.Equalf(t, stagingApp, pushApplications[1], "unexpected staging app")
	require.Equalf(t, prodApp, pushApplications[2], "unexpected prod app")

	require.Equalf(t, appProject, pushProjects[0], "unexpected app project")
}
