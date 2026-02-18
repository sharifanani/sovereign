package ws

import (
	"log"
	"net/http"

	"nhooyr.io/websocket"

	"github.com/sovereign-im/sovereign/server/internal/auth"
	"github.com/sovereign-im/sovereign/server/internal/mls"
	"github.com/sovereign-im/sovereign/server/internal/store"
)

// UpgradeHandler returns an HTTP handler that upgrades connections to WebSocket.
func UpgradeHandler(hub *Hub, maxMessageSize int, authService *auth.Service, st *store.Store, mlsSvc *mls.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"sovereign.v1"},
		})
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		// Verify subprotocol was negotiated.
		if conn.Subprotocol() != "sovereign.v1" {
			conn.Close(websocket.StatusPolicyViolation, "unsupported subprotocol")
			return
		}

		id := connID()
		c := NewConn(id, conn, hub, maxMessageSize, authService, st, mlsSvc)

		log.Printf("New WebSocket connection: %s from %s", id, r.RemoteAddr)

		// Run the connection (blocking).
		c.Run(r.Context())
	}
}
