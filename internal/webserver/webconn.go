package webserver

import (
	"context"

	"github.com/gorilla/websocket"
)

type WebConn struct {
	// The request context for the connection
	ctx context.Context

	// The underlying connection
	conn *websocket.Conn
}
