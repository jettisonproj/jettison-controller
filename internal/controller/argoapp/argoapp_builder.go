package argoapp

import (
	"strings"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
	v1alpha1base "github.com/jettisonproj/jettison-controller/api/v1alpha1/base"
	"github.com/jettisonproj/jettison-controller/internal/gitutil"
)

const (
	appsNamespace     = "argocd"
	configRepoSuffix  = "-argo-configs"
	defaultFinalizer  = "resources-finalizer.argocd.argoproj.io"
	targetRevision    = "HEAD"
	destinationServer = "https://kubernetes.default.svc"
	defaultProject    = "default"
	syncOption        = "CreateNamespace=true"
)

var (
	typeMeta = metav1.TypeMeta{
		Kind:       cdv1.ApplicationSchemaGroupVersionKind.Kind,
		APIVersion: cdv1.SchemeGroupVersion.String(),
	}
	syncPolicy = &cdv1.SyncPolicy{
		Automated: &cdv1.SyncPolicyAutomated{
			Prune:    true,
			SelfHeal: true,
		},
		SyncOptions: []string{syncOption},
	}
)

func BuildArgoApps(flowSteps []v1alpha1base.BaseStep) ([]*cdv1.Application, error) {
	var applications []*cdv1.Application
	for _, flowStep := range flowSteps {
		switch step := flowStep.(type) {
		case *v1alpha1.ArgoCDStep:
			repoOrg, repoName, err := gitutil.GetRepoOrgName(step.RepoUrl)
			if err != nil {
				return nil, err
			}
			repoName = strings.TrimSuffix(repoName, configRepoSuffix)
			appName := repoName + "-" + strings.ReplaceAll(step.RepoPath, "/", "-")

			application := &cdv1.Application{
				TypeMeta: typeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:       appName,
					Namespace:  appsNamespace,
					Finalizers: []string{defaultFinalizer},
				},
				Spec: cdv1.ApplicationSpec{
					Source: &cdv1.ApplicationSource{
						RepoURL:        step.RepoUrl,
						Path:           step.RepoPath,
						TargetRevision: targetRevision,
					},
					Destination: cdv1.ApplicationDestination{
						Server:    destinationServer,
						Namespace: repoOrg,
					},
					Project:    defaultProject,
					SyncPolicy: syncPolicy,
				},
			}
			applications = append(applications, application)
		}
	}
	return applications, nil
}
