package webserver

import (
	"context"
	"fmt"

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
		return fmt.Errorf("failed to get namespace informer for webserver: %s", err)
	}
	return nil
}

func (s *FlowWatcher) OnAdd(obj interface{}, isInInitialList bool) {
	setupLog.Info("got add event", "obj", obj, "isInInitialList", isInInitialList)

	switch resource := obj.(type) {
	case *corev1.Namespace:
		if resource.APIVersion == "" || resource.Kind == "" {
			s.backfillNamespaceSchema(resource)
		}
		s.notify <- corev1.NamespaceList{Items: []corev1.Namespace{*resource}}
	default:
		err := fmt.Errorf("unknown add event type: %T", obj)
		setupLog.Error(err, "unknown type in add event")
	}
}

func (s *FlowWatcher) OnUpdate(oldObj, newObj interface{}) {
	setupLog.Info("got update event", "oldbj", oldObj, "newObj", newObj)

	switch newResource := newObj.(type) {
	case *corev1.Namespace:
		if newResource.APIVersion == "" || newResource.Kind == "" {
			s.backfillNamespaceSchema(newResource)
		}
		s.notify <- corev1.NamespaceList{Items: []corev1.Namespace{*newResource}}
	default:
		err := fmt.Errorf("unknown update event type: %T", newObj)
		setupLog.Error(err, "unknown type in update event")
	}
}

func (s *FlowWatcher) OnDelete(obj interface{}) {
	setupLog.Info("got delete event", "obj", obj)

	switch resource := obj.(type) {
	case *corev1.Namespace:
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
	default:
		err := fmt.Errorf("unknown delete event type: %T", obj)
		setupLog.Error(err, "unknown type in delete event")
	}
}

// Backfill TypeMeta data that has been dropped
// See: https://github.com/kubernetes/client-go/issues/541
func (s *FlowWatcher) backfillNamespaceSchema(namespace *corev1.Namespace) {
	setupLog.Info("backfilling namespace api version kind")
	gvk, err := apiutil.GVKForObject(namespace, s.scheme)
	if err != nil {
		setupLog.Error(err, "unable to get namespace schema info")
		panic(err)
	}
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	namespace.APIVersion = apiVersion
	namespace.Kind = kind
}
