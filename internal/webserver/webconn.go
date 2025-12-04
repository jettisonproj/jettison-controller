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
}
