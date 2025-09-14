package webserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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
	"github.com/jettisonproj/jettison-controller/internal/webserver/argosync"
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
	BindAddress          string
	WorkflowMysqlAddress string
	ArgoCDKey            string
	Client               client.Client
	Cache                cache.Cache
	Scheme               *runtime.Scheme
	flowWatcher          *FlowWatcher
	mysqlWorkflowsMap    map[string]map[string]workflowsv1.Workflow
	syncClient           *argosync.SyncClient
}

// SetupWithManager sets up the webserver with the Manager.
func (s *FlowWebServer) SetupWithManager(mgr ctrl.Manager) error {

	// Load the workflows from mysql for historical runs
	mysqlWorkflows, err := getWorkflowsFromMysql(s.WorkflowMysqlAddress)
	if err != nil {
		return fmt.Errorf(
			"failed to fetch mysql workflows from %s: %s",
			s.WorkflowMysqlAddress,
			err,
		)
	}

	mysqlWorkflowsMap := make(map[string]map[string]workflowsv1.Workflow)
	for _, mysqlWorkflow := range mysqlWorkflows {
		mysqlWorkflowsByName := mysqlWorkflowsMap[mysqlWorkflow.Namespace]
		if mysqlWorkflowsByName == nil {
			mysqlWorkflowsByName = make(map[string]workflowsv1.Workflow)
			mysqlWorkflowsMap[mysqlWorkflow.Namespace] = mysqlWorkflowsByName
		}
		mysqlWorkflowsByName[mysqlWorkflow.Name] = mysqlWorkflow
	}
	s.mysqlWorkflowsMap = mysqlWorkflowsMap

	// Load the argocd key
	argocdCredBytes, err := os.ReadFile(s.ArgoCDKey)
	if err != nil {
		return fmt.Errorf("failed to read ArgoCD key: %s", err)
	}
	s.syncClient, err = argosync.NewSyncClient(strings.TrimSpace(string(argocdCredBytes)))
	if err != nil {
		return fmt.Errorf("failed to create ArgoCD sync client: %s", err)
	}

	// Set up server
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

	mux.HandleFunc("POST /api/v1/namespaces/{namespace}/applications/{name}/sync", s.syncApplication)

	err = mgr.Add(&manager.Server{
		Name:     "jettison server",
		Server:   newServer(mux),
		Listener: serverListener,
	})
	if err != nil {
		return fmt.Errorf("failed to add webserver to manager: %s", err)
	}

	// Set up watcher to send updates
	flowWatcher := &FlowWatcher{
		client:         s.Client,
		cache:          s.Cache,
		scheme:         s.Scheme,
		notify:         make(chan interface{}),
		register:       make(chan *WebConn),
		unregister:     make(chan *WebConn),
		conns:          make(map[*WebConn]bool),
		mysqlWorkflows: mysqlWorkflows,
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
			log.FromContext(ctx).Info(
				"workflow not found in k8s. Checking mysql",
				"namespace", namespace,
				"name", name,
			)
			mysqlWorkflow, found := s.getMysqlWorkflow(namespace, name)
			if found {
				json.NewEncoder(w).Encode(mysqlWorkflow)
				return
			}
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

func (s *FlowWebServer) getMysqlWorkflow(namespace string, name string) (workflowsv1.Workflow, bool) {
	mysqlWorkflowsByName, found := s.mysqlWorkflowsMap[namespace]
	if !found {
		return workflowsv1.Workflow{}, false
	}
	mysqlWorkflow, found := mysqlWorkflowsByName[name]
	return mysqlWorkflow, found
}

// There are many ways to sync an ArgoCD application.
// Ultimately, the top level "operation" field needs to be updated.
// Here are some ways it could be achieved:
//
//  1. Using the ArgoCD API sync api: /api/v1/applications/{name}/sync
//     This requires ArgoCD authentication
//  2. Set the refresh annotations (e.g. argocd.argoproj.io/refresh).
//     The downside is that the operation will be done async
//  3. Manually updating the application using the k8s API
//
// The decision is to go with the ArgoCD API since this seems to handle
// more cases (e.g. sync windows, concurrent updates, retries) and is
// likely the expected way for the sync to happen
func (s *FlowWebServer) syncApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	err := s.syncClient.Sync(ctx, namespace, name)
	if err != nil {
		// todo handle 404 not found error
		log.FromContext(ctx).Error(err, "failed to sync ArgoCD application")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to sync ArgoCD application"}`)
		return
	}

	log.FromContext(ctx).Info(
		"application synced successfully",
		"name", name,
		"namespace", namespace,
	)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"application synced successfully"}`)
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
