package store

import (
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
