package ws

import (
	"log"
	"sync"
)

// Hub manages active WebSocket connections and message routing.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*Conn
	users map[string]*Conn // userID -> authenticated connection

	register   chan *Conn
	unregister chan *Conn
	done       chan struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		conns:      make(map[string]*Conn),
		users:      make(map[string]*Conn),
		register:   make(chan *Conn),
		unregister: make(chan *Conn),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop. It should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.conns[conn.id] = conn
			h.mu.Unlock()
			log.Printf("Connection registered: %s", conn.id)

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.conns[conn.id]; ok {
				delete(h.conns, conn.id)
				if conn.userID != "" {
					delete(h.users, conn.userID)
				}
			}
			h.mu.Unlock()
			log.Printf("Connection unregistered: %s", conn.id)

		case <-h.done:
			return
		}
	}
}

// Stop signals the hub to stop its run loop.
func (h *Hub) Stop() {
	close(h.done)
}

// Register adds a connection to the hub.
func (h *Hub) Register(conn *Conn) {
	h.register <- conn
}

// Unregister removes a connection from the hub.
func (h *Hub) Unregister(conn *Conn) {
	h.unregister <- conn
}

// SetAuthenticated records a connection as authenticated for a user.
func (h *Hub) SetAuthenticated(conn *Conn, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users[userID] = conn
}

// GetConnByUserID returns the authenticated connection for a user, or nil.
func (h *Hub) GetConnByUserID(userID string) *Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.users[userID]
}

// Count returns the number of all active connections.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// AuthenticatedCount returns the number of authenticated connections.
func (h *Hub) AuthenticatedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users)
}
