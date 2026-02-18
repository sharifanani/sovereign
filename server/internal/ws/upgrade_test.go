package ws

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"github.com/sovereign-im/sovereign/server/internal/protocol"
)

func TestUpgradeSubprotocol(t *testing.T) {
	tests := []struct {
		name        string
		subprotocol string
		wantOK      bool
	}{
		{
			name:        "correct subprotocol accepted",
			subprotocol: "sovereign.v1",
			wantOK:      true,
		},
		{
			name:        "wrong subprotocol rejected",
			subprotocol: "wrong.v1",
			wantOK:      false,
		},
		{
			name:        "no subprotocol rejected",
			subprotocol: "",
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			go hub.Run()
			defer hub.Stop()

			handler := UpgradeHandler(hub, 65536, nil)
			server := httptest.NewServer(handler)
			defer server.Close()

			url := "ws" + strings.TrimPrefix(server.URL, "http")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			opts := &websocket.DialOptions{}
			if tt.subprotocol != "" {
				opts.Subprotocols = []string{tt.subprotocol}
			}

			conn, _, err := websocket.Dial(ctx, url, opts)
			if err != nil {
				if tt.wantOK {
					t.Fatalf("Dial failed unexpectedly: %v", err)
				}
				// Dial itself failed — acceptable rejection path
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			if tt.wantOK {
				// Verify the connection works by sending a PING
				// (PING works during auth phase)
				pingPayload, _ := proto.Marshal(&protocol.Ping{Timestamp: 12345})
				env := &protocol.Envelope{
					Type:      protocol.MessageType_PING,
					RequestId: "upgrade-test",
					Payload:   pingPayload,
				}
				sendEnvelope(t, ctx, conn, env)

				resp := readEnvelope(t, ctx, conn)
				if resp.Type != protocol.MessageType_PONG {
					t.Errorf("Response type = %v, want PONG", resp.Type)
				}
				if resp.RequestId != "upgrade-test" {
					t.Errorf("RequestId = %q, want %q", resp.RequestId, "upgrade-test")
				}
			} else {
				// Server should close the connection with StatusPolicyViolation
				_, _, err := conn.Read(ctx)
				if err == nil {
					t.Fatal("Expected connection to be closed for wrong subprotocol")
				}

				if status := websocket.CloseStatus(err); status != websocket.StatusPolicyViolation {
					t.Errorf("Close status = %d, want %d (StatusPolicyViolation)", status, websocket.StatusPolicyViolation)
				}
			}
		})
	}
}
