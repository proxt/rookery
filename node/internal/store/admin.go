package store

import (
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const defaultAdminUsername = "admin"

// EnsureAdmin makes sure an admin login row exists, generating a random
// password on first run. generatedPassword is non-empty only when a new
// password was just generated, so the caller can print it once.
func (s *Store) EnsureAdmin() (generatedPassword string, err error) {
	var exists bool
	err = s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin WHERE id = 1)`).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("store: check admin: %w", err)
	}
	if exists {
		return "", nil
	}

	password, err := randomToken(9)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("store: hash password: %w", err)
	}

	_, err = s.db.Exec(`INSERT INTO admin (id, username, password_hash) VALUES (1, ?, ?)`,
		defaultAdminUsername, string(hash))
	if err != nil {
		return "", fmt.Errorf("store: create admin: %w", err)
	}
	return password, nil
}

// VerifyAdmin reports whether username/password match the stored admin login.
func (s *Store) VerifyAdmin(username, password string) bool {
	var storedUsername, hash string
	err := s.db.QueryRow(`SELECT username, password_hash FROM admin WHERE id = 1`).Scan(&storedUsername, &hash)
	if err != nil {
		return false
	}
	if username != storedUsername {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// AdminUsername returns the current admin username.
func (s *Store) AdminUsername() (string, error) {
	var username string
	err := s.db.QueryRow(`SELECT username FROM admin WHERE id = 1`).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("store: get admin username: %w", err)
	}
	return username, nil
}

// UpdateAdmin changes the admin username and/or password. Pass "" for
// either argument to leave it unchanged.
func (s *Store) UpdateAdmin(newUsername, newPassword string) error {
	if newUsername != "" {
		if _, err := s.db.Exec(`UPDATE admin SET username = ? WHERE id = 1`, newUsername); err != nil {
			return fmt.Errorf("store: update admin username: %w", err)
		}
	}
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("store: hash password: %w", err)
		}
		if _, err := s.db.Exec(`UPDATE admin SET password_hash = ? WHERE id = 1`, string(hash)); err != nil {
			return fmt.Errorf("store: update admin password: %w", err)
		}
	}
	return nil
}

// PublicAddr returns the node's admin-configured public address, used in
// generated rookery:// links.
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

// SetPublicAddr updates the node's public address.
func (s *Store) SetPublicAddr(addr string) error {
	_, err := s.db.Exec(`UPDATE settings SET public_addr = ? WHERE id = 1`, addr)
	if err != nil {
		return fmt.Errorf("store: set public_addr: %w", err)
	}
	return nil
}

// SetPublicAddrIfEmpty sets the public address only if it hasn't been set
// yet, e.g. for a one-time auto-detected default. Reports whether it made a
// change.
func (s *Store) SetPublicAddrIfEmpty(addr string) (bool, error) {
	res, err := s.db.Exec(`UPDATE settings SET public_addr = ? WHERE id = 1 AND public_addr = ''`, addr)
	if err != nil {
		return false, fmt.Errorf("store: set public_addr: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: set public_addr: %w", err)
	}
	return n > 0, nil
}
