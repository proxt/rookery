// Package config loads the node's YAML configuration. Every field has a
// sane default, so the node runs with zero configuration; the file itself
// is optional and only needed to override defaults.
package config

import (
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the node's runtime configuration.
type Config struct {
	ListenAddr           string `yaml:"listen_addr"`
	ICEUDPPort           int    `yaml:"ice_udp_port"`
	DataDir              string `yaml:"data_dir"`
	MaxStreams           int    `yaml:"max_streams_per_session"`
	DialTimeoutSec       int    `yaml:"dial_timeout_seconds"`
	AllowPrivateNet      bool   `yaml:"allow_private_net"`
	BufferedAmountLowKB  int    `yaml:"buffered_amount_low_kb"`
	BufferedAmountHighKB int    `yaml:"buffered_amount_high_kb"`
	LogLevel             string `yaml:"log_level"`
}

// Defaults returns the configuration used for anything not overridden by a
// config file.
func Defaults() Config {
	return Config{
		ListenAddr:           "127.0.0.1:8080",
		ICEUDPPort:           51000,
		DataDir:              "/data",
		MaxStreams:           256,
		DialTimeoutSec:       10,
		AllowPrivateNet:      false,
		BufferedAmountLowKB:  256,
		BufferedAmountHighKB: 1024,
		LogLevel:             "info",
	}
}

// Load reads a Config from a YAML file at path, applying Defaults() for any
// field the file doesn't set. A missing file is not an error — the node
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

// Validate checks that all fields are sane. Since every field has a
// built-in default, this only guards against an explicit override being
// invalid.
func (c *Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return fmt.Errorf("config: listen_addr %q: %w", c.ListenAddr, err)
	}
	if c.ICEUDPPort <= 0 || c.ICEUDPPort > 65535 {
		return fmt.Errorf("config: ice_udp_port must be between 1 and 65535, got %d", c.ICEUDPPort)
	}
	if c.DataDir == "" {
		return fmt.Errorf("config: data_dir must not be empty")
	}
	if c.MaxStreams <= 0 {
		return fmt.Errorf("config: max_streams_per_session must be positive, got %d", c.MaxStreams)
	}
	if c.DialTimeoutSec <= 0 {
		return fmt.Errorf("config: dial_timeout_seconds must be positive, got %d", c.DialTimeoutSec)
	}
	if c.BufferedAmountLowKB <= 0 {
		return fmt.Errorf("config: buffered_amount_low_kb must be positive, got %d", c.BufferedAmountLowKB)
	}
	if c.BufferedAmountHighKB <= c.BufferedAmountLowKB {
		return fmt.Errorf("config: buffered_amount_high_kb (%d) must be greater than buffered_amount_low_kb (%d)", c.BufferedAmountHighKB, c.BufferedAmountLowKB)
	}
	return nil
}
