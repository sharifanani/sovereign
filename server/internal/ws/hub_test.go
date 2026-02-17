package ws

import (
	"testing"
	"time"
)

func TestHubRegisterUnregister(t *testing.T) {
	tests := []struct {
		name       string
		register   []string
		unregister []string
		wantCount  int
	}{
		{
			name:      "single registration",
			register:  []string{"conn-1"},
			wantCount: 1,
		},
		{
			name:      "multiple registrations",
			register:  []string{"conn-1", "conn-2", "conn-3"},
			wantCount: 3,
		},
		{
			name:       "register then unregister one",
			register:   []string{"conn-1", "conn-2"},
			unregister: []string{"conn-1"},
			wantCount:  1,
		},
		{
			name:       "register and unregister all",
			register:   []string{"conn-1", "conn-2"},
			unregister: []string{"conn-1", "conn-2"},
			wantCount:  0,
		},
		{
			name:       "unregister nonexistent connection",
			register:   []string{"conn-1"},
			unregister: []string{"conn-99"},
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewHub()
			go hub.Run()
			defer hub.Stop()

			conns := make(map[string]*Conn)
			for _, id := range tt.register {
				c := &Conn{id: id}
				conns[id] = c
				hub.Register(c)
			}
			time.Sleep(50 * time.Millisecond)

			for _, id := range tt.unregister {
				c, ok := conns[id]
				if !ok {
					c = &Conn{id: id}
				}
				hub.Unregister(c)
			}
			if len(tt.unregister) > 0 {
				time.Sleep(50 * time.Millisecond)
			}

			if got := hub.Count(); got != tt.wantCount {
				t.Errorf("Count() = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestHubStop(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	hub.Stop()

	select {
	case <-done:
		// Run loop terminated successfully
	case <-time.After(time.Second):
		t.Fatal("Hub.Run() did not terminate after Stop()")
	}
}
