// Package config loads and validates the node's YAML configuration.
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
	Secret               string `yaml:"secret"`
	SecretEnv            string `yaml:"secret_env"`
	MaxStreams           int    `yaml:"max_streams_per_session"`
	DialTimeoutSec       int    `yaml:"dial_timeout_seconds"`
	AllowPrivateNet      bool   `yaml:"allow_private_net"`
	BufferedAmountLowKB  int    `yaml:"buffered_amount_low_kb"`
	BufferedAmountHighKB int    `yaml:"buffered_amount_high_kb"`
	LogLevel             string `yaml:"log_level"`
}

// Load reads and validates a Config from a YAML file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if cfg.SecretEnv != "" {
		val, ok := os.LookupEnv(cfg.SecretEnv)
		if !ok {
			return nil, fmt.Errorf("config: secret_env %q not set", cfg.SecretEnv)
		}
		cfg.Secret = val
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that all required fields are present and sane.
func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("config: listen_addr is required")
	}
	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return fmt.Errorf("config: listen_addr %q: %w", c.ListenAddr, err)
	}
	if c.ICEUDPPort <= 0 || c.ICEUDPPort > 65535 {
		return fmt.Errorf("config: ice_udp_port must be between 1 and 65535, got %d", c.ICEUDPPort)
	}
	if c.Secret == "" {
		return fmt.Errorf("config: secret is required")
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
