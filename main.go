package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type app struct {
	conf            *config
	ctx             context.Context               // Background ^c context.
	activeStreams   map[string]context.CancelFunc // Allows control of the stream subprocesses.
	activeStreamsMu sync.Mutex                    // Protects activeStreams.
	videoDir        string                        // Where to store the video files.
	wg              *sync.WaitGroup               // Tracks active stream subprocesses.
}

func main() {
	conf, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	videoDir, err := ensureVideoDir()
	if err != nil {
		slog.Error("Failed to check/create video dir", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown propagates through to HTTP server and stream subprocesses.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	exitCh := make(chan struct{})

	var wg sync.WaitGroup

	a := &app{
		conf:          conf,
		videoDir:      videoDir,
		ctx:           ctx,
		activeStreams: map[string]context.CancelFunc{},
		wg:            &wg,
	}

	// Initialize activeStreams with all streams disabled.
	a.activeStreamsMu.Lock()
	for name := range conf.Streams {
		a.activeStreams[name] = nil
	}
	a.activeStreamsMu.Unlock()

	// On ^c, cancel the parent context and wait for everybody to exit.
	// If the waiting times out, systemd will kill us eventually.
	go func() {
		<-sigCh
		slog.Info("Received signal, shutting down...")
		cancel()
		close(exitCh)
	}()

	// HTTP server to control the streams.
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	// Authentication middleware.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+conf.Secret {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Post("/stream/{streamID}/enable", a.handleEnableStream)
	r.Post("/stream/{streamID}/disable", a.handleDisableStream)
	r.Get("/stream/{streamID}/status", a.handleGetStreamStatus)
	r.Get("/streams", a.handleListStreams)

	srv := &http.Server{
		Addr:    "127.0.0.1:31930",
		Handler: r,
	}
	// If starting up the HTTP server fails, kill everything else by faking ^c from above.
	go func() {
		slog.Info("Starting HTTP server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if ctx.Err() == context.Canceled {
				return
			}
			slog.Error("Failed to start HTTP server", "error", err)
			sigCh <- os.Interrupt
		}
	}()
	// Graceful shutdown handler for the HTTP server.
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	// To quit, there needs to be both nobody on the wait group, and ^c must have be present.
	<-exitCh
	wg.Wait()
}
