package webserver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

var (
	setupLog = ctrl.Log.WithName("setupwebserver")
)

// FlowWatcher watches Flow and related objects
type FlowWatcher struct {
	Cache cache.Cache
}

func (s *FlowWatcher) setupWatcher() error {
	ctx := context.Background()

	// send namespace events
	namespaceEventHandler := NamespaceEventHandler{}
	namespaceInformer, err := s.Cache.GetInformer(ctx, &corev1.Namespace{})
	if err != nil {
		return fmt.Errorf("failed to get namespace informer for webserver: %s", err)
	}
	_, err = namespaceInformer.AddEventHandler(&namespaceEventHandler)
	if err != nil {
		return fmt.Errorf("failed to get namespace informer for webserver: %s", err)
	}
	return nil
}

type NamespaceEventHandler struct{}

func (e *NamespaceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	setupLog.Info("got add event", "obj", obj, "isInInitialList", isInInitialList)
}

func (e *NamespaceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	setupLog.Info("got update event", "oldbj", oldObj, "newObj", newObj)
}

func (e *NamespaceEventHandler) OnDelete(obj interface{}) {
	setupLog.Info("got delete event", "obj", obj)
}
