package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue   = "Rookery"
)

// setAutoStart adds or removes the HKCU Run key entry that launches this
// executable at login.
func setAutoStart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("autostart: open run key: %w", err)
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("autostart: remove run value: %w", err)
		}
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("autostart: resolve executable path: %w", err)
	}

	if err := key.SetStringValue(runValue, `"`+exePath+`"`); err != nil {
		return fmt.Errorf("autostart: set run value: %w", err)
	}
	return nil
}
