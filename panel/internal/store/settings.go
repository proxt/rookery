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
