// Package config loads the panel's YAML configuration. Every field has a
// sane default; the file itself is optional.
package config

import (
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the panel's runtime configuration.
type Config struct {
	ListenAddr         string `yaml:"listen_addr"`
	DataDir            string `yaml:"data_dir"`
	SessionTokenTTLMin int    `yaml:"session_token_ttl_minutes"`
	LogLevel           string `yaml:"log_level"`
}

// Defaults returns the configuration used for anything not overridden by a
// config file.
func Defaults() Config {
	return Config{
		ListenAddr:         "127.0.0.1:8090",
		DataDir:            "/data",
		SessionTokenTTLMin: 360, // 6h — client subscription refresh interval must stay well under this
		LogLevel:           "info",
	}
}

// Load reads a Config from a YAML file at path, applying Defaults() for any
// field the file doesn't set. A missing file is not an error — the panel
// just runs on defaults.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &cfg, cfg.Validate()
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that all fields are sane.
func (c *Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return fmt.Errorf("config: listen_addr %q: %w", c.ListenAddr, err)
	}
	if c.DataDir == "" {
		return fmt.Errorf("config: data_dir must not be empty")
	}
	if c.SessionTokenTTLMin <= 0 {
		return fmt.Errorf("config: session_token_ttl_minutes must be positive, got %d", c.SessionTokenTTLMin)
	}
	return nil
}
