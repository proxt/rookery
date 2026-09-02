package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const defaultAdminUsername = "admin"

// Admin is one panel operator login.
type Admin struct {
	ID        string
	Username  string
	CreatedAt time.Time
}

// ErrAdminExists is returned by CreateAdmin when the username is taken.
var ErrAdminExists = errors.New("store: admin username already exists")

// EnsureAdmin makes sure at least one admin login exists, generating a
// random password for a default "admin" account on first run.
// generatedPassword is non-empty only when a new account was just created,
// so the caller can print it once.
func (s *Store) EnsureAdmin() (generatedPassword string, err error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return "", fmt.Errorf("store: count admins: %w", err)
	}
	if count > 0 {
		return "", nil
	}

	password, err := randomToken(9)
	if err != nil {
		return "", err
	}
	if _, err := s.CreateAdmin(defaultAdminUsername, password); err != nil {
		return "", err
	}
	return password, nil
}

// ListAdmins returns every admin login, oldest first.
func (s *Store) ListAdmins() ([]Admin, error) {
	rows, err := s.db.Query(`SELECT id, username, created_at FROM admins ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("store: list admins: %w", err)
	}
	defer rows.Close()

	var out []Admin
	for rows.Next() {
		var a Admin
		var createdAt string
		if err := rows.Scan(&a.ID, &a.Username, &createdAt); err != nil {
			return nil, fmt.Errorf("store: scan admin: %w", err)
		}
		a.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("store: parse created_at: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAdmin adds a new admin login.
func (s *Store) CreateAdmin(username, password string) (Admin, error) {
	id, err := randomToken(8)
	if err != nil {
		return Admin{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Admin{}, fmt.Errorf("store: hash password: %w", err)
	}

	a := Admin{ID: id, Username: username, CreatedAt: time.Now().UTC()}
	_, err = s.db.Exec(`INSERT INTO admins (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		a.ID, a.Username, string(hash), a.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		if isUniqueConstraintErr(err) {
			return Admin{}, ErrAdminExists
		}
		return Admin{}, fmt.Errorf("store: create admin: %w", err)
	}
	return a, nil
}

// DeleteAdmin removes an admin login by ID.
func (s *Store) DeleteAdmin(id string) error {
	res, err := s.db.Exec(`DELETE FROM admins WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete admin: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete admin: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AdminCount reports how many admin logins exist.
func (s *Store) AdminCount() (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count admins: %w", err)
	}
	return count, nil
}

// VerifyAdmin reports whether username/password match a stored admin login,
// returning that admin's ID if so.
func (s *Store) VerifyAdmin(username, password string) (string, bool) {
	var id, hash string
	err := s.db.QueryRow(`SELECT id, password_hash FROM admins WHERE username = ?`, username).Scan(&id, &hash)
	if err != nil {
		return "", false
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", false
	}
	return id, true
}

// GetAdmin returns one admin by ID.
func (s *Store) GetAdmin(id string) (Admin, error) {
	var a Admin
	var createdAt string
	err := s.db.QueryRow(`SELECT id, username, created_at FROM admins WHERE id = ?`, id).Scan(&a.ID, &a.Username, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrNotFound
	}
	if err != nil {
		return Admin{}, fmt.Errorf("store: get admin: %w", err)
	}
	a.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Admin{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return a, nil
}

// UpdateAdminPassword changes one admin's password.
func (s *Store) UpdateAdminPassword(id, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: hash password: %w", err)
	}
	res, err := s.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?`, string(hash), id)
	if err != nil {
		return fmt.Errorf("store: update admin password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update admin password: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT")
}
