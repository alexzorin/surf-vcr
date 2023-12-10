package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"time"
)

func runStream(ctx context.Context, name string, conf streamConfig, videoDir string) error {
	defer slog.Info("Stopped stream", "name", name)

	for {
		videoName := fmt.Sprintf("%s-%s.ts", name, time.Now().Format(time.RFC3339))
		videoPath := filepath.Join(videoDir, videoName)

		slog.Info("Starting stream", "name", name, "source", conf.Source,
			"destination", videoPath)

		cmd := exec.CommandContext(ctx, "gst-launch-1.0",
			conf.Source, "!", "hlsdemux", "!", "filesink", "location="+videoPath)
		cmd.Dir = videoDir

		if err := cmd.Run(); err != nil {
			if ctx.Err() == context.Canceled {
				return nil
			}
			slog.Warn("Stream exited with error, will restart", "name", name, "error", err)
		} else {
			slog.Warn("Stream exited without error, will restart", "name", name)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
			continue
		}
	}
}
