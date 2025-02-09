package webserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
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
}

// SetupWithManager sets up the webserver with the Manager.
func (s *FlowWebServer) SetupWithManager(mgr ctrl.Manager) error {
	serverListener, err := net.Listen("tcp", s.BindAddress)
	if err != nil {
		return fmt.Errorf("webserver failed to listen: %s", err)
	}

	err = mgr.Add(&manager.Server{
		Name:     "jettison server",
		Server:   newServer(s),
		Listener: serverListener,
	})
	if err != nil {
		return fmt.Errorf("failed to add webserver to manager: %s", err)
	}

	flowWatcher := &FlowWatcher{
		Cache: s.Cache,
	}
	err = flowWatcher.setupWatcher()
	if err != nil {
		return fmt.Errorf("failed to set up watcher for webserver: %s", err)
	}

	return nil
}

func (s *FlowWebServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestLog := log.FromContext(ctx)

	websocketConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		requestLog.Error(err, "failed to get websocket connection for request")
		return
	}
	defer websocketConnection.Close()

	// send namespaces
	namespaces := &corev1.NamespaceList{}
	err = s.Client.List(ctx, namespaces)
	if err != nil {
		requestLog.Error(err, "failed to get namespaces for websocket")
		return
	}
	err = websocketConnection.WriteJSON(namespaces)
	if err != nil {
		requestLog.Error(err, "failed to send namespaces for websocket")
		return
	}

	// send flows
	flows := &v1alpha1.FlowList{}
	err = s.Client.List(ctx, flows)
	if err != nil {
		requestLog.Error(err, "failed to get flows for websocket")
		return
	}
	err = websocketConnection.WriteJSON(flows)
	if err != nil {
		requestLog.Error(err, "failed to send flows for websocket")
		return
	}

	// handle messages
	for {
		messageType, messageBytes, err := websocketConnection.ReadMessage()
		if err != nil {
			requestLog.Error(err, "failed to read from websocket")
			break
		}
		requestLog.Info("receive from websocket", "message", string(messageBytes))
		err = websocketConnection.WriteMessage(messageType, messageBytes)
		if err != nil {
			requestLog.Error(err, "failed to write to websocket")
			break
		}
	}
}

// Ideally, could use package:
//
//	"sigs.k8s.io/controller-runtime/pkg/internal/httpserver"
//
// Which is used for health probe server. However, it's an internal
// package, so add it here instead. See:
//
// https://github.com/kubernetes-sigs/controller-runtime/blob/main/pkg/internal/httpserver/server.go
//
// Returns a new server with sane defaults.
func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		MaxHeaderBytes:    1 << 20,
		IdleTimeout:       90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadHeaderTimeout: 32 * time.Second,
	}
}
