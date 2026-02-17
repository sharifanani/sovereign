package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sovereign-im/sovereign/server/internal/config"
	"github.com/sovereign-im/sovereign/server/internal/ws"
	"github.com/sovereign-im/sovereign/server/web"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := config.DefaultConfig()
	log.Printf("Sovereign server starting on %s", cfg.ListenAddr)

	hub := ws.NewHub()
	go hub.Run()

	mux := http.NewServeMux()

	// WebSocket endpoint.
	mux.Handle("/ws", ws.UpgradeHandler(hub, cfg.MaxMessageSize))

	// Embedded admin UI.
	adminFS, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatalf("Failed to create admin UI filesystem: %v", err)
	}
	mux.Handle("/admin/", http.StripPrefix("/admin/", http.FileServer(http.FS(adminFS))))

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	// Start server in a goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Server listening on %s", cfg.ListenAddr)

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("Received signal %s, shutting down...", sig)

	hub.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
