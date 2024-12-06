package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

func (r *FlowReconciler) setCondition(
	ctx context.Context,
	req ctrl.Request,
	flow *v1alpha1.Flow,
	condition metav1.Condition,
) error {
	changed := meta.SetStatusCondition(&flow.Status.Conditions, condition)
	if !changed {
		return nil
	}

	if err := r.Status().Update(ctx, flow); err != nil {
		return err
	}

	// Let's re-fetch the Flow Custom Resource after updating the status
	// so that we have the latest state of the resource on the cluster and we will avoid
	// raising the error "the object has been modified, please apply
	// your changes to the latest version and try again" which would re-trigger the reconciliation
	// if we try to update it again in the following operations
	if err := r.Get(ctx, req.NamespacedName, flow); err != nil {
		return err
	}
	return nil
}
