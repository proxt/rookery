// Package config loads and validates the client's YAML configuration.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the client's runtime configuration.
type Config struct {
	SubscriptionName string `yaml:"subscription_name"`
	PanelAddr        string `yaml:"panel_addr"`
	Token            string `yaml:"token"`
	TokenEnv         string `yaml:"token_env"`
	// NodeID picks a specific node from the subscription's list. Empty
	// means "the first node the panel returns".
	NodeID               string `yaml:"node_id"`
	SOCKSAddr            string `yaml:"socks_addr"`
	BufferedAmountLowKB  int    `yaml:"buffered_amount_low_kb"`
	BufferedAmountHighKB int    `yaml:"buffered_amount_high_kb"`
	ReconnectMaxBackoffS int    `yaml:"reconnect_max_backoff_seconds"`
	StartMinimized       bool   `yaml:"start_minimized"`
	AutoStart            bool   `yaml:"auto_start"`
	SystemWide           bool   `yaml:"system_wide"`
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

	if cfg.TokenEnv != "" {
		val, ok := os.LookupEnv(cfg.TokenEnv)
		if !ok {
			return nil, fmt.Errorf("config: token_env %q not set", cfg.TokenEnv)
		}
		cfg.Token = val
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that all required fields are present and sane.
func (c *Config) Validate() error {
	if c.PanelAddr == "" {
		return fmt.Errorf("config: panel_addr is required")
	}
	if c.Token == "" {
		return fmt.Errorf("config: token is required")
	}
	if c.SOCKSAddr == "" {
		return fmt.Errorf("config: socks_addr is required")
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
