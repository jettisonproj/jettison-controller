package webserver

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
)

type WebConn struct {
	// The request Context for the connection
	ctx context.Context

	// The request CancelCauseFunc for the connection
	cancelCauseFunc context.CancelCauseFunc

	// The log with the request context
	log logr.Logger

	// The underlying connection
	conn *websocket.Conn

	// The write channel for the connection
	// Used to synchronize writes. Enforces one writer
	writeChan chan any

	// Whether this is an auxiliary websocket connection. For example,
	// a secondary websocket to stream container logs in a new tab
	isAuxWs bool
}
