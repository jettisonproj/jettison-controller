package webserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	cdv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	rolloutsv1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	workflowsv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1alpha1 "github.com/jettisonproj/jettison-controller/api/v1alpha1"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // todo productionize
		},
	}
)

// FlowWebServer serves Flow and related objects
type FlowWebServer struct {
	BindAddress string
	Client      client.Client
	Cache       cache.Cache
	Scheme      *runtime.Scheme
	flowWatcher *FlowWatcher
}

// SetupWithManager sets up the webserver with the Manager.
func (s *FlowWebServer) SetupWithManager(mgr ctrl.Manager) error {
	serverListener, err := net.Listen("tcp", s.BindAddress)
	if err != nil {
		return fmt.Errorf("webserver failed to listen: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", s.handleWebsocket)
	mux.HandleFunc("GET /api/v1/namespaces/{name}", s.handleNamespace)
	mux.HandleFunc("GET /api/v1/namespaces/{namespace}/flows/{name}", s.handleFlow)
	mux.HandleFunc("GET /api/v1/namespaces/{namespace}/applications/{name}", s.handleApplication)
	mux.HandleFunc("GET /api/v1/namespaces/{namespace}/rollouts/{name}", s.handleRollout)
	mux.HandleFunc("GET /api/v1/namespaces/{namespace}/workflows/{name}", s.handleWorkflow)
	err = mgr.Add(&manager.Server{
		Name:     "jettison server",
		Server:   newServer(mux),
		Listener: serverListener,
	})
	if err != nil {
		return fmt.Errorf("failed to add webserver to manager: %s", err)
	}

	flowWatcher := &FlowWatcher{
		client:     s.Client,
		cache:      s.Cache,
		scheme:     s.Scheme,
		notify:     make(chan interface{}),
		register:   make(chan *WebConn),
		unregister: make(chan *WebConn),
		conns:      make(map[*WebConn]bool),
	}
	err = flowWatcher.setupWatcher()
	if err != nil {
		return fmt.Errorf("failed to set up watcher for webserver: %s", err)
	}
	s.flowWatcher = flowWatcher
	return nil
}

func (s *FlowWebServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get websocket connection for request")
		return
	}
	s.flowWatcher.register <- &WebConn{ctx: ctx, conn: conn}
}

func (s *FlowWebServer) handleNamespace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	namespace := &corev1.Namespace{}
	name := r.PathValue("name")
	objectKey := client.ObjectKey{
		Namespace: name,
		Name:      name,
	}
	err := s.Client.Get(ctx, objectKey, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("namespace not found", "name", name)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"namespace not found"}`)
			return
		}
		log.FromContext(ctx).Error(err, "failed to fetch namespace", "name", name)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to fetch namespace"}`)
		return
	}
	json.NewEncoder(w).Encode(namespace)
}

func (s *FlowWebServer) handleFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	flow := &v1alpha1.Flow{}
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	objectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := s.Client.Get(ctx, objectKey, flow)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("flow not found", "namespace", namespace, "name", name)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"flow not found"}`)
			return
		}
		log.FromContext(ctx).Error(err, "failed to fetch flow", "namespace", namespace, "name", name)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to fetch flow"}`)
		return
	}
	json.NewEncoder(w).Encode(flow)
}

func (s *FlowWebServer) handleApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	application := &cdv1.Application{}
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	objectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := s.Client.Get(ctx, objectKey, application)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("application not found", "namespace", namespace, "name", name)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"application not found"}`)
			return
		}
		log.FromContext(ctx).Error(err, "failed to fetch application", "namespace", namespace, "name", name)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to fetch application"}`)
		return
	}
	json.NewEncoder(w).Encode(application)
}

func (s *FlowWebServer) handleRollout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	rollout := &rolloutsv1.Rollout{}
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	objectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := s.Client.Get(ctx, objectKey, rollout)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("rollout not found", "namespace", namespace, "name", name)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"rollout not found"}`)
			return
		}
		log.FromContext(ctx).Error(err, "failed to fetch rollout", "namespace", namespace, "name", name)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to fetch rollout"}`)
		return
	}
	json.NewEncoder(w).Encode(rollout)
}

func (s *FlowWebServer) handleWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	workflow := &workflowsv1.Workflow{}
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	objectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := s.Client.Get(ctx, objectKey, workflow)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Info("workflow not found", "namespace", namespace, "name", name)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"workflow not found"}`)
			return
		}
		log.FromContext(ctx).Error(err, "failed to fetch workflow", "namespace", namespace, "name", name)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to fetch workflow"}`)
		return
	}
	json.NewEncoder(w).Encode(workflow)
}

// Returns a new server with sane defaults. Based on internal package:
// https://github.com/kubernetes-sigs/controller-runtime/blob/main/pkg/internal/httpserver/server.go
func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		MaxHeaderBytes:    1 << 20,
		IdleTimeout:       90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadHeaderTimeout: 32 * time.Second,
	}
}
