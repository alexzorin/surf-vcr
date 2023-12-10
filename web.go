package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *app) handleEnableStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")

	a.activeStreamsMu.Lock()
	defer a.activeStreamsMu.Unlock()

	streamCancelFunc, ok := a.activeStreams[streamID]
	if !ok {
		http.Error(w, "No such stream", http.StatusNotFound)
		return
	}
	if streamCancelFunc != nil {
		http.Error(w, "Stream already enabled", http.StatusBadRequest)
		return
	}

	streamCtx, streamCancelFunc := context.WithCancel(a.ctx)
	a.activeStreams[streamID] = streamCancelFunc

	a.wg.Add(1)

	go func() {
		defer a.wg.Done()
		if err := runStream(streamCtx, streamID, a.conf.Streams[streamID], a.videoDir); err != nil {
			slog.Error("Failed to run stream", "name", streamID, "error", err)
		}
		a.activeStreamsMu.Lock()
		a.activeStreams[streamID] = nil
		a.activeStreamsMu.Unlock()
	}()

	fmt.Fprintf(w, "OK. Started the %s stream", streamID)
}

func (a *app) handleDisableStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")

	a.activeStreamsMu.Lock()
	defer a.activeStreamsMu.Unlock()

	streamCancelFunc, ok := a.activeStreams[streamID]
	if !ok {
		http.Error(w, "No such stream", http.StatusNotFound)
		return
	}
	if streamCancelFunc == nil {
		http.Error(w, "Stream already disabled", http.StatusBadRequest)
		return
	}

	streamCancelFunc()

	fmt.Fprintf(w, "OK. Stopped the %s stream", streamID)
}

func (a *app) handleGetStreamStatus(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")

	a.activeStreamsMu.Lock()
	streamCancelFunc, ok := a.activeStreams[streamID]
	a.activeStreamsMu.Unlock()

	if !ok {
		http.Error(w, "No such stream", http.StatusNotFound)
		return
	}
	if streamCancelFunc == nil {
		fmt.Fprintf(w, "Stream %s is disabled", streamID)
	} else {
		fmt.Fprintf(w, "Stream %s is enabled", streamID)
	}
}

func (a *app) handleListStreams(w http.ResponseWriter, r *http.Request) {
	statuses := map[string]bool{}

	a.activeStreamsMu.Lock()
	for name, cancelFunc := range a.activeStreams {
		statuses[name] = cancelFunc != nil
	}
	a.activeStreamsMu.Unlock()

	for name, enabled := range statuses {
		if enabled {
			fmt.Fprintf(w, "Stream %s is enabled\n", name)
		} else {
			fmt.Fprintf(w, "Stream %s is disabled\n", name)
		}
	}
}
