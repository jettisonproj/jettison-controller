package appbuilder

import (
	"fmt"
	"strings"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
	"github.com/jettisonproj/jettison-controller/internal/gitutil"
)

const (
	argocdNamespace   = "argocd"
	configRepoSuffix  = "-argo-configs"
	defaultFinalizer  = "resources-finalizer.argocd.argoproj.io"
	targetRevision    = "HEAD"
	destinationServer = "https://kubernetes.default.svc"
	syncOption        = "CreateNamespace=true"
)

var (
	projectTypeMeta = metav1.TypeMeta{
		Kind:       cdv1.AppProjectSchemaGroupVersionKind.Kind,
		APIVersion: cdv1.SchemeGroupVersion.String(),
	}
	applicationTypeMeta = metav1.TypeMeta{
		Kind:       cdv1.ApplicationSchemaGroupVersionKind.Kind,
		APIVersion: cdv1.SchemeGroupVersion.String(),
	}
	retryStrategy = &cdv1.RetryStrategy{
		// Use the latest revision on retry instead of the initial one
		Refresh: true,
	}
)

func BuildArgoApps(flowSteps []v1alpha1base.BaseStep) ([]*cdv1.AppProject, []*cdv1.Application, error) {
	var projects []*cdv1.AppProject
	var applications []*cdv1.Application

	// Use a set to dedup
	projectNames := make(map[string]bool)
	appNames := make(map[string]bool)

	for _, flowStep := range flowSteps {
		switch step := flowStep.(type) {
		case *v1alpha1.ArgoCDStep:
			repoOrg, repoName, err := gitutil.GetRepoOrgName(step.RepoUrl)
			if err != nil {
				return nil, nil, err
			}

			if !projectNames[repoOrg] {
				projectNames[repoOrg] = true

				sourceRepoGlob := gitutil.GetSourceRepoGlob(repoOrg)

				project := &cdv1.AppProject{
					TypeMeta: projectTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						// The repo org and project name match
						Name:      repoOrg,
						Namespace: argocdNamespace,
					},
					Spec: cdv1.AppProjectSpec{
						SourceRepos: []string{sourceRepoGlob},
						Destinations: []cdv1.ApplicationDestination{
							{
								Server: destinationServer,
								// The repo org and target namespace match
								Namespace: repoOrg,
							},
						},
						// The repo org and application namespace match
						SourceNamespaces: []string{repoOrg},
						ClusterResourceWhitelist: []cdv1.ClusterResourceRestrictionItem{
							{
								Group: "*",
								Kind:  "*",
								Name:  fmt.Sprintf("%s-*", repoOrg),
							},
						},
					},
				}
				projects = append(projects, project)
			}

			appName := GetAppName(repoName, step.RepoPath)

			if !appNames[appName] {
				appNames[appName] = true

				stepEnabled := step.PausedReason == nil || *step.PausedReason == ""
				application := &cdv1.Application{
					TypeMeta: applicationTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						Name: appName,
						// The repo org and application namespace match
						Namespace:  repoOrg,
						Finalizers: []string{defaultFinalizer},
					},
					Spec: cdv1.ApplicationSpec{
						Source: &cdv1.ApplicationSource{
							RepoURL:        step.RepoUrl,
							Path:           step.RepoPath,
							TargetRevision: targetRevision,
						},
						Destination: cdv1.ApplicationDestination{
							Server: destinationServer,
							// The repo org and target namespace match
							Namespace: repoOrg,
						},
						// The repo org and project match
						Project: repoOrg,
						SyncPolicy: &cdv1.SyncPolicy{
							Automated: &cdv1.SyncPolicyAutomated{
								Prune:    true,
								SelfHeal: true,
								Enabled:  &stepEnabled,
							},
							SyncOptions: []string{syncOption},
							Retry:       retryStrategy,
						},
					},
				}
				applications = append(applications, application)
			}
		}
	}
	return projects, applications, nil
}

func GetAppName(repoName string, repoPath string) string {
	return fmt.Sprintf(
		"%s-%s",
		strings.TrimSuffix(repoName, configRepoSuffix),
		strings.ReplaceAll(repoPath, "/", "-"),
	)
}
