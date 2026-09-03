package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// PublicAddr returns the panel's admin-configured public address, used to
// build sub_url for generated subscriptions.
func (s *Store) PublicAddr() (string, error) {
	var addr string
	err := s.db.QueryRow(`SELECT public_addr FROM settings WHERE id = 1`).Scan(&addr)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get public_addr: %w", err)
	}
	return addr, nil
}

// SetPublicAddr updates the panel's public address.
func (s *Store) SetPublicAddr(addr string) error {
	_, err := s.db.Exec(`UPDATE settings SET public_addr = ? WHERE id = 1`, addr)
	if err != nil {
		return fmt.Errorf("store: set public_addr: %w", err)
	}
	return nil
}

// AutoUpdateEnabled reports whether the panel should trigger Watchtower's
// on-demand update check on its own periodic sweep (see
// server.triggerAutoUpdatePeriodically). Defaults to true so existing
// deployments keep updating automatically unless an admin turns it off.
func (s *Store) AutoUpdateEnabled() (bool, error) {
	var enabled bool
	err := s.db.QueryRow(`SELECT auto_update_enabled FROM settings WHERE id = 1`).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: get auto_update_enabled: %w", err)
	}
	return enabled, nil
}

// SetAutoUpdateEnabled updates whether the panel triggers automatic updates.
func (s *Store) SetAutoUpdateEnabled(enabled bool) error {
	_, err := s.db.Exec(`UPDATE settings SET auto_update_enabled = ? WHERE id = 1`, enabled)
	if err != nil {
		return fmt.Errorf("store: set auto_update_enabled: %w", err)
	}
	return nil
}
