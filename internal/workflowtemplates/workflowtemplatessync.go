package workflowtemplates

import (
	"context"
	"time"

	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	workflowTemplateSyncRetryInterval = 5 * time.Second
)

type WorkflowTemplateSyncer struct {
	Client ctrlclient.Client
}

func (s *WorkflowTemplateSyncer) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			log.Info("skip workflow template sync attempt due to cancelled context")
			return nil
		}

		log.Info("syncing cluster workflow templates")
		cicdTemplate := &workflowsv1.ClusterWorkflowTemplate{
			TypeMeta:   CICDTemplate.TypeMeta,
			ObjectMeta: CICDTemplate.ObjectMeta,
		}

		op, err := ctrl.CreateOrUpdate(
			ctx,
			s.Client,
			cicdTemplate,
			func() error {
				cicdTemplate.Spec = CICDTemplate.Spec
				return nil
			},
		)

		if ctx.Err() != nil {
			log.Info("skip workflow template sync result due to cancelled context")
			return nil
		}

		if err != nil {
			log.Error(err, "unable to sync cluster workflow templates. Retrying...")
			time.Sleep(workflowTemplateSyncRetryInterval)
			continue
		}

		log.Info("synced cluster workflow templates", "operation", op)
		break
	}
	return nil
}

func (s *WorkflowTemplateSyncer) NeedLeaderElection() bool {
	return true
}
