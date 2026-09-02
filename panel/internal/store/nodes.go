package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Node is a registered relay server. APIKey authenticates both the node's
// own heartbeat/report calls back to the panel, and the session tokens the
// panel issues for it.
type Node struct {
	ID         string
	Name       string
	Address    string // public base URL clients POST /session to
	APIKey     string
	Tags       string
	Enabled    bool
	LastSeenAt string // RFC3339Nano, or "" if never
	CreatedAt  time.Time
}

// ListNodes returns all nodes, newest first.
func (s *Store) ListNodes() ([]Node, error) {
	rows, err := s.db.Query(`SELECT id, name, address, api_key, tags, enabled, last_seen_at, created_at FROM nodes ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list nodes: %w", err)
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
		return nil, fmt.Errorf("store: list nodes: %w", err)
	}
	return out, nil
}

// GetNode returns the node with the given ID.
func (s *Store) GetNode(id string) (Node, error) {
	row := s.db.QueryRow(`SELECT id, name, address, api_key, tags, enabled, last_seen_at, created_at FROM nodes WHERE id = ?`, id)
	n, err := scanNodeRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	return n, err
}

// CreateNode registers a new node with a random ID and API key.
func (s *Store) CreateNode(name, address, tags string) (Node, error) {
	id, err := randomToken(6)
	if err != nil {
		return Node{}, err
	}
	apiKey, err := randomToken(24)
	if err != nil {
		return Node{}, err
	}

	n := Node{ID: id, Name: name, Address: address, APIKey: apiKey, Tags: tags, Enabled: true, CreatedAt: time.Now().UTC()}
	_, err = s.db.Exec(`INSERT INTO nodes (id, name, address, api_key, tags, enabled, last_seen_at, created_at)
		VALUES (?, ?, ?, ?, ?, 1, '', ?)`,
		n.ID, n.Name, n.Address, n.APIKey, n.Tags, n.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Node{}, fmt.Errorf("store: create node: %w", err)
	}
	return n, nil
}

// UpdateNode changes a node's name/address/tags/enabled flag.
func (s *Store) UpdateNode(id, name, address, tags string, enabled bool) error {
	res, err := s.db.Exec(`UPDATE nodes SET name = ?, address = ?, tags = ?, enabled = ? WHERE id = ?`,
		name, address, tags, enabled, id)
	if err != nil {
		return fmt.Errorf("store: update node: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update node: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteNode removes a node. Its subscription assignments cascade-delete.
func (s *Store) DeleteNode(id string) error {
	res, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete node: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete node: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchNode records that a node's heartbeat was just received.
func (s *Store) TouchNode(id string) error {
	res, err := s.db.Exec(`UPDATE nodes SET last_seen_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("store: touch node: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: touch node: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// NodeByAPIKey looks up a node by ID, verifying apiKey matches — used to
// authenticate a node's own heartbeat/report calls.
func (s *Store) NodeByAPIKey(id, apiKey string) (Node, error) {
	n, err := s.GetNode(id)
	if err != nil {
		return Node{}, err
	}
	if n.APIKey != apiKey {
		return Node{}, ErrNotFound
	}
	return n, nil
}

func scanNode(rows *sql.Rows) (Node, error) {
	var n Node
	var enabled int
	var createdAt string
	if err := rows.Scan(&n.ID, &n.Name, &n.Address, &n.APIKey, &n.Tags, &enabled, &n.LastSeenAt, &createdAt); err != nil {
		return Node{}, fmt.Errorf("store: scan node: %w", err)
	}
	n.Enabled = enabled != 0
	var err error
	n.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Node{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return n, nil
}

func scanNodeRow(row *sql.Row) (Node, error) {
	var n Node
	var enabled int
	var createdAt string
	if err := row.Scan(&n.ID, &n.Name, &n.Address, &n.APIKey, &n.Tags, &enabled, &n.LastSeenAt, &createdAt); err != nil {
		return Node{}, err
	}
	n.Enabled = enabled != 0
	var err error
	n.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Node{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return n, nil
}
