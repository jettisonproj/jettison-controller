package webserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	err = mgr.Add(&manager.Server{
		Name:     "jettison server",
		Server:   newServer(s),
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

func (s *FlowWebServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get websocket connection for request")
		return
	}
	s.flowWatcher.register <- &WebConn{ctx: ctx, conn: conn}
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
