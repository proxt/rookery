package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User is a panel identity: a name, a subscription token (embedded in the
// rookery:// link), an enabled flag, an optional active window, and the set
// of nodes it grants access to.
type User struct {
	ID           string
	Name         string
	Token        string
	Enabled      bool
	StartsAt     string // RFC3339Nano, or "" for no lower bound
	ExpiresAt    string // RFC3339Nano, or "" for never
	LastActiveAt string // RFC3339Nano, or "" if never reported traffic
	CreatedAt    time.Time
}

// ErrNotFound is returned when a lookup or delete targets an unknown ID.
var ErrNotFound = errors.New("store: not found")

// ListUsers returns all users, newest first.
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, name, token, enabled, starts_at, expires_at, last_active_at, created_at
		FROM users ORDER BY created_at DESC`)
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
	row := s.db.QueryRow(`SELECT id, name, token, enabled, starts_at, expires_at, last_active_at, created_at
		FROM users WHERE id = ?`, id)
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// GetUserByToken looks up a user by their subscription token — used by the
// client-facing /sub/{token} endpoint.
func (s *Store) GetUserByToken(token string) (User, error) {
	row := s.db.QueryRow(`SELECT id, name, token, enabled, starts_at, expires_at, last_active_at, created_at
		FROM users WHERE token = ?`, token)
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// CreateUser adds a new user with a random ID and subscription token.
func (s *Store) CreateUser(name string) (User, error) {
	id, err := randomToken(8)
	if err != nil {
		return User{}, err
	}
	token, err := randomToken(20)
	if err != nil {
		return User{}, err
	}

	u := User{ID: id, Name: name, Token: token, Enabled: true, CreatedAt: time.Now().UTC()}
	_, err = s.db.Exec(`INSERT INTO users (id, name, token, enabled, starts_at, expires_at, created_at)
		VALUES (?, ?, ?, 1, '', '', ?)`,
		u.ID, u.Name, u.Token, u.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return User{}, fmt.Errorf("store: create user: %w", err)
	}
	return u, nil
}

// UpdateUser changes a user's name/enabled flag/active window. startsAt and
// expiresAt are RFC3339Nano, or "" for no bound.
func (s *Store) UpdateUser(id, name string, enabled bool, startsAt, expiresAt string) error {
	res, err := s.db.Exec(`UPDATE users SET name = ?, enabled = ?, starts_at = ?, expires_at = ? WHERE id = ?`,
		name, enabled, startsAt, expiresAt, id)
	if err != nil {
		return fmt.Errorf("store: update user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update user: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchUserActivity records that traffic was just reported for a user —
// what the admin UI's "online" indicator is based on. Best-effort: called
// from RecordTraffic, where a user not existing (deleted mid-session, say)
// shouldn't fail the traffic report itself.
func (s *Store) TouchUserActivity(id string) error {
	_, err := s.db.Exec(`UPDATE users SET last_active_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("store: touch user activity: %w", err)
	}
	return nil
}

// DeleteUser removes a user by ID. Its node assignments cascade-delete; its
// stat samples are removed explicitly (stat_samples has no FK, to keep
// historical global totals intact even after the node that earned them is
// gone — but a deleted user's own data is gone with them).
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
	if _, err := s.db.Exec(`DELETE FROM stat_samples WHERE user_id = ?`, id); err != nil {
		return fmt.Errorf("store: delete user stats: %w", err)
	}
	return nil
}

// SetUserNodes replaces the set of nodes a user has access to.
func (s *Store) SetUserNodes(userID string, nodeIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: set user nodes: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM user_nodes WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("store: set user nodes: %w", err)
	}
	for _, nodeID := range nodeIDs {
		if _, err := tx.Exec(`INSERT INTO user_nodes (user_id, node_id) VALUES (?, ?)`, userID, nodeID); err != nil {
			return fmt.Errorf("store: set user nodes: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: set user nodes: %w", err)
	}
	return nil
}

// ListUserNodes returns the nodes a user has access to, skipping any that
// are disabled.
func (s *Store) ListUserNodes(userID string) ([]Node, error) {
	rows, err := s.db.Query(`SELECT n.id, n.name, n.address, n.api_key, n.tags, n.enabled, n.last_seen_at, n.created_at
		FROM nodes n
		JOIN user_nodes un ON un.node_id = n.id
		WHERE un.user_id = ? AND n.enabled = 1
		ORDER BY n.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list user nodes: %w", err)
	}
	defer rows.Close()

	var out []Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list user nodes: %w", err)
	}
	return out, nil
}

func scanUser(rows *sql.Rows) (User, error) {
	var u User
	var enabled int
	var createdAt string
	if err := rows.Scan(&u.ID, &u.Name, &u.Token, &enabled, &u.StartsAt, &u.ExpiresAt, &u.LastActiveAt, &createdAt); err != nil {
		return User{}, fmt.Errorf("store: scan user: %w", err)
	}
	u.Enabled = enabled != 0
	var err error
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return User{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return u, nil
}

func scanUserRow(row *sql.Row) (User, error) {
	var u User
	var enabled int
	var createdAt string
	if err := row.Scan(&u.ID, &u.Name, &u.Token, &enabled, &u.StartsAt, &u.ExpiresAt, &u.LastActiveAt, &createdAt); err != nil {
		return User{}, err
	}
	u.Enabled = enabled != 0
	var err error
	u.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return User{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return u, nil
}
