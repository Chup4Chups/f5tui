package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host      string `yaml:"host"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	Insecure  bool   `yaml:"insecure"`
	Partition string `yaml:"partition"`
}

// DefaultPath returns the preferred config location:
// $XDG_CONFIG_HOME/f5tui/config.yaml (falling back to ~/.config/f5tui/config.yaml).
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "f5tui", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "f5tui", "config.yaml")
}

// Load reads a config file. A missing file at the default path is not an error.
func Load(path string, explicit bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}
