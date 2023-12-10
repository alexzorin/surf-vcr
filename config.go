package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type config struct {
	Secret  string                  `toml:"secret"`
	Streams map[string]streamConfig `toml:"streams"`
}

type streamConfig struct {
	Source string `toml:"source"`
}

func loadConfig() (*config, error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(confDir, "surf-vcr.toml")

	var conf config

	if _, err := toml.DecodeFile(path, &conf); err != nil {
		return nil, fmt.Errorf("failed to load config from %q: %w", path, err)
	}

	return &conf, nil
}

func ensureVideoDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	videoDir := filepath.Join(cacheDir, "surf-vcr")

	if _, stat := os.Stat(videoDir); os.IsNotExist(stat) {
		if err := os.MkdirAll(videoDir, 0755); err != nil {
			return "", fmt.Errorf("couldn't create video dir at %q: %w", videoDir, err)
		}
	}

	return videoDir, nil
}
