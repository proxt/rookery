// Package config loads and validates the client's YAML configuration.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the client's runtime configuration.
type Config struct {
	NodeAddr             string `yaml:"node_addr"`
	SOCKSAddr            string `yaml:"socks_addr"`
	Secret               string `yaml:"secret"`
	SecretEnv            string `yaml:"secret_env"`
	BufferedAmountLowKB  int    `yaml:"buffered_amount_low_kb"`
	BufferedAmountHighKB int    `yaml:"buffered_amount_high_kb"`
	ReconnectMaxBackoffS int    `yaml:"reconnect_max_backoff_seconds"`
	StartMinimized       bool   `yaml:"start_minimized"`
	AutoStart            bool   `yaml:"auto_start"`
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
	if c.NodeAddr == "" {
		return fmt.Errorf("config: node_addr is required")
	}
	if c.SOCKSAddr == "" {
		return fmt.Errorf("config: socks_addr is required")
	}
	if c.Secret == "" {
		return fmt.Errorf("config: secret is required")
	}
	if c.BufferedAmountLowKB <= 0 {
		return fmt.Errorf("config: buffered_amount_low_kb must be positive, got %d", c.BufferedAmountLowKB)
	}
	if c.BufferedAmountHighKB <= c.BufferedAmountLowKB {
		return fmt.Errorf("config: buffered_amount_high_kb (%d) must be greater than buffered_amount_low_kb (%d)", c.BufferedAmountHighKB, c.BufferedAmountLowKB)
	}
	if c.ReconnectMaxBackoffS <= 0 {
		return fmt.Errorf("config: reconnect_max_backoff_seconds must be positive, got %d", c.ReconnectMaxBackoffS)
	}
	return nil
}
