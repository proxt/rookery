package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile is one saved connection: everything a rookery:// link carries,
// plus a stable local ID so the UI can reference it across renames.
type Profile struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	NodeAddr string `json:"nodeAddr" yaml:"node_addr"`
	UserID   string `json:"userId" yaml:"user_id"`
	Secret   string `json:"secret" yaml:"secret"`
}

// AppSettings is everything the GUI persists: the profile list plus
// general, profile-independent settings.
type AppSettings struct {
	Profiles        []Profile `json:"profiles" yaml:"profiles"`
	ActiveProfileID string    `json:"activeProfileId" yaml:"active_profile_id"`
	SOCKSPort       int       `json:"socksPort" yaml:"socks_port"`
	AutoStart       bool      `json:"autoStart" yaml:"auto_start"`
	StartMinimized  bool      `json:"startMinimized" yaml:"start_minimized"`
	SystemWide      bool      `json:"systemWide" yaml:"system_wide"`
}

// defaultSOCKSPort is used whenever no SOCKS port has been configured yet.
const defaultSOCKSPort = 1080

func defaultAppSettings() AppSettings {
	return AppSettings{Profiles: []Profile{}, SOCKSPort: defaultSOCKSPort, SystemWide: true}
}

// loadSettings reads settings from path. A missing file is not an error —
// it returns defaults instead.
func loadSettings(path string) (AppSettings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultAppSettings(), nil
	}
	if err != nil {
		return AppSettings{}, fmt.Errorf("read settings: %w", err)
	}

	s := defaultAppSettings()
	if err := yaml.Unmarshal(data, &s); err != nil {
		return AppSettings{}, fmt.Errorf("parse settings: %w", err)
	}
	// A YAML file with an empty/absent "profiles:" key unmarshals to a nil
	// slice, which encodes as JSON null — the frontend always expects an
	// array it can call .length/.find on.
	if s.Profiles == nil {
		s.Profiles = []Profile{}
	}
	return s, nil
}

func saveSettings(path string, s AppSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// ActiveProfile returns the currently selected profile, if any.
func (s AppSettings) ActiveProfile() (Profile, bool) {
	for _, p := range s.Profiles {
		if p.ID == s.ActiveProfileID {
			return p, true
		}
	}
	return Profile{}, false
}
