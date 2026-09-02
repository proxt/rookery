package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User is a panel-level identity a subscription belongs to.
type User struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// ErrNotFound is returned when a lookup or delete targets an unknown ID.
var ErrNotFound = errors.New("store: not found")

// ListUsers returns all users, newest first.
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list users: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list users: %w", err)
	}
	return out, nil
}

// GetUser returns the user with the given ID.
func (s *Store) GetUser(id string) (User, error) {
	var u User
	var createdAt string
	err := s.db.QueryRow(`SELECT id, name, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("store: get user: %w", err)
	}
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return User{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return u, nil
}

// CreateUser adds a new user with a random ID.
func (s *Store) CreateUser(name string) (User, error) {
	id, err := randomToken(8)
	if err != nil {
		return User{}, err
	}

	u := User{ID: id, Name: name, CreatedAt: time.Now().UTC()}
	_, err = s.db.Exec(`INSERT INTO users (id, name, created_at) VALUES (?, ?, ?)`,
		u.ID, u.Name, u.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return User{}, fmt.Errorf("store: create user: %w", err)
	}
	return u, nil
}

// DeleteUser removes a user by ID. Its subscriptions cascade-delete.
func (s *Store) DeleteUser(id string) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete user: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(rows *sql.Rows) (User, error) {
	var u User
	var createdAt string
	if err := rows.Scan(&u.ID, &u.Name, &createdAt); err != nil {
		return User{}, fmt.Errorf("store: scan user: %w", err)
	}
	var err error
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return User{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return u, nil
}
