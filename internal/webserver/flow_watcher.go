package webserver

import (
	"context"
	"fmt"
	"slices"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	rolloutsv1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

var (
	setupLog = ctrl.Log.WithName("setupwebserver")
)

// FlowWatcher watches Flow and related objects
type FlowWatcher struct {
	// Initial resources are fetched from the ctrlclient and sent to the connecting websocket
	client ctrlclient.Client

	// Updates to resources are fetched from the ctrlcache and sent to the connected websocket
	cache ctrlcache.Cache

	// Scheme to retrieve additional schema details
	scheme *runtime.Scheme

	// Sends notifications to the websockets
	notify chan interface{}

	// Registers a websocket
	register chan *WebConn

	// Unregisters a websocket
	unregister chan *WebConn

	// Keeps track of registered websockets
	conns map[*WebConn]bool

	// MySQL workflows are sent as initial resources
	mysqlWorkflows []workflowsv1.Workflow
}

func (s *FlowWatcher) run() {
	for {
		select {
		case conn := <-s.register:
			s.registerConn(conn)
		case conn := <-s.unregister:
			s.unregisterConn(conn)
		case message := <-s.notify:
			for conn := range s.conns {
				s.notifyConn(conn, message)
			}
		}
	}
}

func (s *FlowWatcher) registerConn(conn *WebConn) {
	ctx := conn.ctx
	connLog := log.FromContext(ctx)
	connLog.Info("registering connection")

	// Send the initial resources
	// send namespaces
	namespaces := &corev1.NamespaceList{}
	err := s.client.List(ctx, namespaces)
	if err != nil {
		connLog.Error(err, "failed to get namespaces for websocket")
		err = conn.conn.WriteJSON(newWebError(
			fmt.Sprintf("failed to get namespaces: %s", err),
		))
		if err != nil {
			connLog.Error(err, "failed to send error message after failing to get namespaces")
		}
		s.unregisterConn(conn)
		return
	}
	err = conn.conn.WriteJSON(namespaces)
	if err != nil {
		connLog.Error(err, "failed to send namespaces for websocket")
		s.unregisterConn(conn)
		return
	}

	// send flows
	flows := &v1alpha1.FlowList{}
	err = s.client.List(ctx, flows)
	if err != nil {
		connLog.Error(err, "failed to get flows for websocket")
		err = conn.conn.WriteJSON(newWebError(
			fmt.Sprintf("failed to get flows: %s", err),
		))
		if err != nil {
			connLog.Error(err, "failed to send error message after failing to get flows")
		}
		s.unregisterConn(conn)
		return
	}
	err = conn.conn.WriteJSON(flows)
	if err != nil {
		connLog.Error(err, "failed to send flows for websocket")
		s.unregisterConn(conn)
		return
	}

	// send applications
	applications := &cdv1.ApplicationList{}
	err = s.client.List(ctx, applications)
	if err != nil {
		connLog.Error(err, "failed to get applications for websocket")
		err = conn.conn.WriteJSON(newWebError(
			fmt.Sprintf("failed to get applications: %s", err),
		))
		if err != nil {
			connLog.Error(err, "failed to send error message after failing to get applications")
		}
		s.unregisterConn(conn)
		return
	}
	err = conn.conn.WriteJSON(applications)
	if err != nil {
		connLog.Error(err, "failed to send applications for websocket")
		s.unregisterConn(conn)
		return
	}

	// send rollouts
	rollouts := &rolloutsv1.RolloutList{}
	err = s.client.List(ctx, rollouts)
	if err != nil {
		connLog.Error(err, "failed to get rollouts for websocket")
		err = conn.conn.WriteJSON(newWebError(
			fmt.Sprintf("failed to get rollouts: %s", err),
		))
		if err != nil {
			connLog.Error(err, "failed to send error message after failing to get rollouts")
		}
		s.unregisterConn(conn)
		return
	}
	err = conn.conn.WriteJSON(rollouts)
	if err != nil {
		connLog.Error(err, "failed to send rollouts for websocket")
		s.unregisterConn(conn)
		return
	}

	// send workflows
	workflows := &workflowsv1.WorkflowList{}
	err = s.client.List(ctx, workflows)
	if err != nil {
		connLog.Error(err, "failed to get workflows for websocket")
		err = conn.conn.WriteJSON(newWebError(
			fmt.Sprintf("failed to get workflows: %s", err),
		))
		if err != nil {
			connLog.Error(err, "failed to send error message after failing to get workflows")
		}
		s.unregisterConn(conn)
		return
	}
	workflows.Items = slices.Concat(s.mysqlWorkflows, workflows.Items)
	err = conn.conn.WriteJSON(workflows)
	if err != nil {
		connLog.Error(err, "failed to send workflows for websocket")
		s.unregisterConn(conn)
		return
	}

	// Register the connection to send updates
	s.conns[conn] = true

	go s.readConn(conn)
}

func (s *FlowWatcher) readConn(conn *WebConn) {
	defer func() {
		s.unregister <- conn
	}()
	ctx := conn.ctx
	connLog := log.FromContext(ctx)
	for {
		messageType, message, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				connLog.Error(err, "unexpected close error")
				panic(err)
			}
			connLog.Info("connection exited cleanly")
			return
		}
		err = fmt.Errorf("unexpected message type %d and message %s", messageType, string(message))
		connLog.Error(err, "unexpected message read")
	}
}

func (s *FlowWatcher) unregisterConn(conn *WebConn) {
	ctx := conn.ctx
	connLog := log.FromContext(ctx)
	connLog.Info("unregistering connection")

	delete(s.conns, conn)
	err := conn.conn.Close()
	if err != nil {
		connLog.Error(err, "error closing connection")
	}
}

func (s *FlowWatcher) notifyConn(conn *WebConn, message interface{}) {
	err := conn.conn.WriteJSON(message)
	if err != nil {
		ctx := conn.ctx
		connLog := log.FromContext(ctx)
		connLog.Error(err, "failed to notify conn. Closing...")
		s.unregisterConn(conn)
	}
}

func (s *FlowWatcher) setupWatcher() error {
	go s.run()

	ctx := context.Background()

	// send namespace events
	namespaceInformer, err := s.cache.GetInformer(ctx, &corev1.Namespace{})
	if err != nil {
		return fmt.Errorf("failed to get namespace informer for webserver: %s", err)
	}
	_, err = namespaceInformer.AddEventHandler(s)
	if err != nil {
		return fmt.Errorf("failed to init namespace informer for webserver: %s", err)
	}

	// send flow events
	flowInformer, err := s.cache.GetInformer(ctx, &v1alpha1.Flow{})
	if err != nil {
		return fmt.Errorf("failed to get flow informer for webserver: %s", err)
	}
	_, err = flowInformer.AddEventHandler(s)
	if err != nil {
		return fmt.Errorf("failed to init flow informer for webserver: %s", err)
	}

	// send application events
	applicationInformer, err := s.cache.GetInformer(ctx, &cdv1.Application{})
	if err != nil {
		return fmt.Errorf("failed to get application informer for webserver: %s", err)
	}
	_, err = applicationInformer.AddEventHandler(s)
	if err != nil {
		return fmt.Errorf("failed to init application informer for webserver: %s", err)
	}

	// send rollout events
	rolloutInformer, err := s.cache.GetInformer(ctx, &rolloutsv1.Rollout{})
	if err != nil {
		return fmt.Errorf("failed to get rollout informer for webserver: %s", err)
	}
	_, err = rolloutInformer.AddEventHandler(s)
	if err != nil {
		return fmt.Errorf("failed to init rollout informer for webserver: %s", err)
	}

	// send workflow events
	workflowInformer, err := s.cache.GetInformer(ctx, &workflowsv1.Workflow{})
	if err != nil {
		return fmt.Errorf("failed to get workflow informer for webserver: %s", err)
	}
	_, err = workflowInformer.AddEventHandler(s)
	if err != nil {
		return fmt.Errorf("failed to init workflow informer for webserver: %s", err)
	}
	return nil
}

func (s *FlowWatcher) OnAdd(obj interface{}, isInInitialList bool) {
	switch resource := obj.(type) {
	case *corev1.Namespace:
		setupLog.Info(
			"got add event for Namespace",
			"name", resource.Name,
			"isInInitialList", isInInitialList,
		)
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillNamespaceSchema(resource)
		}
		s.notify <- corev1.NamespaceList{Items: []corev1.Namespace{*resource}}
	case *v1alpha1.Flow:
		setupLog.Info(
			"got add event for Flow",
			"namespace", resource.Namespace,
			"name", resource.Name,
			"isInInitialList", isInInitialList,
		)
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillFlowSchema(resource)
		}
		s.notify <- v1alpha1.FlowList{Items: []v1alpha1.Flow{*resource}}
	case *cdv1.Application:
		setupLog.Info(
			"got add event for Application",
			"namespace", resource.Namespace,
			"name", resource.Name,
			"isInInitialList", isInInitialList,
		)
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillApplicationSchema(resource)
		}
		s.notify <- cdv1.ApplicationList{Items: []cdv1.Application{*resource}}
	case *rolloutsv1.Rollout:
		setupLog.Info(
			"got add event for Rollout",
			"namespace", resource.Namespace,
			"name", resource.Name,
			"isInInitialList", isInInitialList,
		)
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillRolloutSchema(resource)
		}
		s.notify <- rolloutsv1.RolloutList{Items: []rolloutsv1.Rollout{*resource}}
	case *workflowsv1.Workflow:
		setupLog.Info(
			"got add event for Rollout",
			"namespace", resource.Namespace,
			"name", resource.Name,
			"isInInitialList", isInInitialList,
		)
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillWorkflowSchema(resource)
		}
		s.notify <- workflowsv1.WorkflowList{Items: []workflowsv1.Workflow{*resource}}
	default:
		err := fmt.Errorf("unknown add event type: %T", obj)
		setupLog.Error(err, "unknown type in add event")
	}
}

func (s *FlowWatcher) OnUpdate(oldObj, newObj interface{}) {
	switch newResource := newObj.(type) {
	case *corev1.Namespace:
		setupLog.Info(
			"got update event for Namespace",
			"name", newResource.Name,
		)
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillNamespaceSchema(newResource)
		}
		s.notify <- corev1.NamespaceList{Items: []corev1.Namespace{*newResource}}
	case *v1alpha1.Flow:
		setupLog.Info(
			"got update event for Flow",
			"namespace", newResource.Namespace,
			"name", newResource.Name,
		)
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillFlowSchema(newResource)
		}
		s.notify <- v1alpha1.FlowList{Items: []v1alpha1.Flow{*newResource}}
	case *cdv1.Application:
		setupLog.Info(
			"got update event for Application",
			"namespace", newResource.Namespace,
			"name", newResource.Name,
		)
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillApplicationSchema(newResource)
		}
		s.notify <- cdv1.ApplicationList{Items: []cdv1.Application{*newResource}}
	case *rolloutsv1.Rollout:
		setupLog.Info(
			"got update event for Rollout",
			"namespace", newResource.Namespace,
			"name", newResource.Name,
		)
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillRolloutSchema(newResource)
		}
		s.notify <- rolloutsv1.RolloutList{Items: []rolloutsv1.Rollout{*newResource}}
	case *workflowsv1.Workflow:
		setupLog.Info(
			"got update event for Workflow",
			"namespace", newResource.Namespace,
			"name", newResource.Name,
		)
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillWorkflowSchema(newResource)
		}
		s.notify <- workflowsv1.WorkflowList{Items: []workflowsv1.Workflow{*newResource}}
	default:
		err := fmt.Errorf("unknown update event type: %T", newObj)
		setupLog.Error(err, "unknown type in update event")
	}
}

func (s *FlowWatcher) OnDelete(obj interface{}) {
	switch resource := obj.(type) {
	case *corev1.Namespace:
		setupLog.Info(
			"got delete event for Namespace",
			"name", resource.Name,
		)

		// Add a distinguishing delete key
		annotationKey := fmt.Sprintf("%s/watcher-event-type", v1alpha1.GroupVersion.Identifier())
		annotationVal := "delete"
		if resource.Annotations == nil {
			resource.Annotations = map[string]string{annotationKey: annotationVal}
		} else {
			resource.Annotations[annotationKey] = annotationVal
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillNamespaceSchema(resource)
		}
		s.notify <- corev1.NamespaceList{Items: []corev1.Namespace{*resource}}
	case *v1alpha1.Flow:
		setupLog.Info(
			"got delete event for Flow",
			"namespace", resource.Namespace,
			"name", resource.Name,
		)

		// Add a distinguishing delete key
		annotationKey := fmt.Sprintf("%s/watcher-event-type", v1alpha1.GroupVersion.Identifier())
		annotationVal := "delete"
		if resource.Annotations == nil {
			resource.Annotations = map[string]string{annotationKey: annotationVal}
		} else {
			resource.Annotations[annotationKey] = annotationVal
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillFlowSchema(resource)
		}
		s.notify <- v1alpha1.FlowList{Items: []v1alpha1.Flow{*resource}}
	case *cdv1.Application:
		setupLog.Info(
			"got delete event for Application",
			"namespace", resource.Namespace,
			"name", resource.Name,
		)

		// Add a distinguishing delete key
		annotationKey := fmt.Sprintf("%s/watcher-event-type", v1alpha1.GroupVersion.Identifier())
		annotationVal := "delete"
		if resource.Annotations == nil {
			resource.Annotations = map[string]string{annotationKey: annotationVal}
		} else {
			resource.Annotations[annotationKey] = annotationVal
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillApplicationSchema(resource)
		}
		s.notify <- cdv1.ApplicationList{Items: []cdv1.Application{*resource}}
	case *rolloutsv1.Rollout:
		setupLog.Info(
			"got delete event for Rollout",
			"namespace", resource.Namespace,
			"name", resource.Name,
		)

		// Add a distinguishing delete key
		annotationKey := fmt.Sprintf("%s/watcher-event-type", v1alpha1.GroupVersion.Identifier())
		annotationVal := "delete"
		if resource.Annotations == nil {
			resource.Annotations = map[string]string{annotationKey: annotationVal}
		} else {
			resource.Annotations[annotationKey] = annotationVal
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillRolloutSchema(resource)
		}
		s.notify <- rolloutsv1.RolloutList{Items: []rolloutsv1.Rollout{*resource}}
	case *workflowsv1.Workflow:
		setupLog.Info(
			"got delete event for Workflow",
			"namespace", resource.Namespace,
			"name", resource.Name,
		)
		// Add a distinguishing delete key
		annotationKey := fmt.Sprintf("%s/watcher-event-type", v1alpha1.GroupVersion.Identifier())
		annotationVal := "delete"
		if resource.Annotations == nil {
			resource.Annotations = map[string]string{annotationKey: annotationVal}
		} else {
			resource.Annotations[annotationKey] = annotationVal
		}

		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillWorkflowSchema(resource)
		}
		s.notify <- workflowsv1.WorkflowList{Items: []workflowsv1.Workflow{*resource}}
	default:
		err := fmt.Errorf("unknown delete event type: %T", obj)
		setupLog.Error(err, "unknown type in delete event")
	}
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillNamespaceSchema(namespace *corev1.Namespace) {
	setupLog.Info("backfilling namespace metadata")
	gvk, err := apiutil.GVKForObject(namespace, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get namespace schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	namespace.APIVersion = apiVersion
	namespace.Kind = kind
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillFlowSchema(flow *v1alpha1.Flow) {
	setupLog.Info("backfilling flow metadata")
	gvk, err := apiutil.GVKForObject(flow, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get flow schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	flow.APIVersion = apiVersion
	flow.Kind = kind
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillApplicationSchema(application *cdv1.Application) {
	setupLog.Info("backfilling application metadata")
	gvk, err := apiutil.GVKForObject(application, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get application schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	application.APIVersion = apiVersion
	application.Kind = kind
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillRolloutSchema(rollout *rolloutsv1.Rollout) {
	setupLog.Info("backfilling rollout metadata")
	gvk, err := apiutil.GVKForObject(rollout, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get rollout schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	rollout.APIVersion = apiVersion
	rollout.Kind = kind
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillWorkflowSchema(workflow *workflowsv1.Workflow) {
	setupLog.Info("backfilling workflow metadata")
	gvk, err := apiutil.GVKForObject(workflow, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get rollout schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	workflow.APIVersion = apiVersion
	workflow.Kind = kind
}
