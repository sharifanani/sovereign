package ws

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/sovereign-im/sovereign/server/internal/protocol"
)

// Conn wraps a WebSocket connection with read/write pumps.
type Conn struct {
	id     string
	ws     *websocket.Conn
	hub    *Hub
	send   chan []byte
	once   sync.Once
	cancel context.CancelFunc

	maxMessageSize int64
}

// NewConn creates a new Conn.
func NewConn(id string, ws *websocket.Conn, hub *Hub, maxMessageSize int) *Conn {
	return &Conn{
		id:             id,
		ws:             ws,
		hub:            hub,
		send:           make(chan []byte, 256),
		maxMessageSize: int64(maxMessageSize),
	}
}

// Run starts the read and write pumps. It blocks until the connection is closed.
func (c *Conn) Run(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)

	c.hub.Register(c)
	defer c.hub.Unregister(c)

	c.ws.SetReadLimit(c.maxMessageSize)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.writePump(ctx)
	}()

	go func() {
		defer wg.Done()
		c.readPump(ctx)
	}()

	wg.Wait()
	c.ws.Close(websocket.StatusNormalClosure, "")
}

// readPump reads messages from the WebSocket and processes them.
func (c *Conn) readPump(ctx context.Context) {
	defer c.close()

	for {
		typ, data, err := c.ws.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				log.Printf("[%s] Connection closed normally", c.id)
			} else {
				log.Printf("[%s] Read error: %v", c.id, err)
			}
			return
		}

		if typ != websocket.MessageBinary {
			log.Printf("[%s] Received non-binary message, closing", c.id)
			c.ws.Close(websocket.StatusUnsupportedData, "binary frames only")
			return
		}

		var env protocol.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			log.Printf("[%s] Failed to unmarshal envelope: %v", c.id, err)
			c.sendError(&env, 3001, "Invalid message format", false)
			continue
		}

		c.handleEnvelope(&env)
	}
}

// writePump writes messages from the send channel to the WebSocket.
func (c *Conn) writePump(ctx context.Context) {
	defer c.close()

	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.ws.Write(ctx, websocket.MessageBinary, data); err != nil {
				log.Printf("[%s] Write error: %v", c.id, err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleEnvelope processes a received envelope.
func (c *Conn) handleEnvelope(env *protocol.Envelope) {
	switch env.Type {
	case protocol.MessageType_PING:
		c.handlePing(env)

	case protocol.MessageType_ERROR:
		log.Printf("[%s] Received error message, discarding", c.id)

	default:
		// Phase A: echo the envelope back to the sender.
		c.echoEnvelope(env)
	}
}

// handlePing responds to a PING with a PONG echoing the timestamp.
func (c *Conn) handlePing(env *protocol.Envelope) {
	var ping protocol.Ping
	if err := proto.Unmarshal(env.Payload, &ping); err != nil {
		log.Printf("[%s] Failed to unmarshal ping: %v", c.id, err)
		c.sendError(env, 3001, "Invalid ping payload", false)
		return
	}

	pong := &protocol.Pong{
		Timestamp: ping.Timestamp,
	}

	pongPayload, err := proto.Marshal(pong)
	if err != nil {
		log.Printf("[%s] Failed to marshal pong: %v", c.id, err)
		return
	}

	resp := &protocol.Envelope{
		Type:      protocol.MessageType_PONG,
		RequestId: env.RequestId,
		Payload:   pongPayload,
	}

	c.sendEnvelope(resp)
}

// echoEnvelope sends the envelope back to the sender.
func (c *Conn) echoEnvelope(env *protocol.Envelope) {
	c.sendEnvelope(env)
}

// sendEnvelope serializes and queues an envelope for sending.
func (c *Conn) sendEnvelope(env *protocol.Envelope) {
	data, err := proto.Marshal(env)
	if err != nil {
		log.Printf("[%s] Failed to marshal envelope: %v", c.id, err)
		return
	}

	select {
	case c.send <- data:
	default:
		log.Printf("[%s] Send buffer full, dropping message", c.id)
	}
}

// sendError sends an error envelope to the client.
func (c *Conn) sendError(origEnv *protocol.Envelope, code int32, message string, fatal bool) {
	errMsg := &protocol.Error{
		Code:    code,
		Message: message,
		Fatal:   fatal,
	}

	payload, err := proto.Marshal(errMsg)
	if err != nil {
		log.Printf("[%s] Failed to marshal error: %v", c.id, err)
		return
	}

	requestID := ""
	if origEnv != nil {
		requestID = origEnv.RequestId
	}

	env := &protocol.Envelope{
		Type:      protocol.MessageType_ERROR,
		RequestId: requestID,
		Payload:   payload,
	}

	c.sendEnvelope(env)
}

// close cancels the connection context, closing both pumps.
func (c *Conn) close() {
	c.once.Do(func() {
		c.cancel()
	})
}

// connID generates a unique connection ID.
func connID() string {
	return fmt.Sprintf("conn-%d", time.Now().UnixNano())
}
