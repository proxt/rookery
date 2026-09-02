package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Subscription grants a user access to a set of nodes via a single opaque
// token, embedded in the rookery:// link handed to them.
type Subscription struct {
	ID        string
	UserID    string
	Name      string
	Token     string
	Enabled   bool
	ExpiresAt string // RFC3339Nano, or "" for never
	CreatedAt time.Time
}

// ListSubscriptions returns all subscriptions, newest first.
func (s *Store) ListSubscriptions() ([]Subscription, error) {
	return s.querySubscriptions(`SELECT id, user_id, name, token, enabled, expires_at, created_at
		FROM subscriptions ORDER BY created_at DESC`)
}

// ListSubscriptionsByUser returns a user's subscriptions, newest first.
func (s *Store) ListSubscriptionsByUser(userID string) ([]Subscription, error) {
	return s.querySubscriptions(`SELECT id, user_id, name, token, enabled, expires_at, created_at
		FROM subscriptions WHERE user_id = ? ORDER BY created_at DESC`, userID)
}

func (s *Store) querySubscriptions(query string, args ...any) ([]Subscription, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list subscriptions: %w", err)
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list subscriptions: %w", err)
	}
	return out, nil
}

// GetSubscription returns the subscription with the given ID.
func (s *Store) GetSubscription(id string) (Subscription, error) {
	row := s.db.QueryRow(`SELECT id, user_id, name, token, enabled, expires_at, created_at
		FROM subscriptions WHERE id = ?`, id)
	sub, err := scanSubscriptionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return sub, err
}

// GetSubscriptionByToken looks up a subscription by its public token — used
// by the client-facing /sub/{token} endpoint.
func (s *Store) GetSubscriptionByToken(token string) (Subscription, error) {
	row := s.db.QueryRow(`SELECT id, user_id, name, token, enabled, expires_at, created_at
		FROM subscriptions WHERE token = ?`, token)
	sub, err := scanSubscriptionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return sub, err
}

// CreateSubscription adds a new subscription for userID with a random ID
// and token.
func (s *Store) CreateSubscription(userID, name string) (Subscription, error) {
	id, err := randomToken(8)
	if err != nil {
		return Subscription{}, err
	}
	token, err := randomToken(20)
	if err != nil {
		return Subscription{}, err
	}

	sub := Subscription{ID: id, UserID: userID, Name: name, Token: token, Enabled: true, CreatedAt: time.Now().UTC()}
	_, err = s.db.Exec(`INSERT INTO subscriptions (id, user_id, name, token, enabled, expires_at, created_at)
		VALUES (?, ?, ?, ?, 1, '', ?)`,
		sub.ID, sub.UserID, sub.Name, sub.Token, sub.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Subscription{}, fmt.Errorf("store: create subscription: %w", err)
	}
	return sub, nil
}

// UpdateSubscription changes a subscription's name/enabled flag/expiry.
// expiresAt is RFC3339Nano, or "" for never.
func (s *Store) UpdateSubscription(id, name string, enabled bool, expiresAt string) error {
	res, err := s.db.Exec(`UPDATE subscriptions SET name = ?, enabled = ?, expires_at = ? WHERE id = ?`,
		name, enabled, expiresAt, id)
	if err != nil {
		return fmt.Errorf("store: update subscription: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update subscription: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSubscription removes a subscription. Its node assignments and stat
// samples cascade-delete (stat_samples has no FK but is cleaned up here).
func (s *Store) DeleteSubscription(id string) error {
	res, err := s.db.Exec(`DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete subscription: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete subscription: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	if _, err := s.db.Exec(`DELETE FROM stat_samples WHERE subscription_id = ?`, id); err != nil {
		return fmt.Errorf("store: delete subscription stats: %w", err)
	}
	return nil
}

// SetSubscriptionNodes replaces the set of nodes a subscription grants
// access to.
func (s *Store) SetSubscriptionNodes(subscriptionID string, nodeIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: set subscription nodes: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM subscription_nodes WHERE subscription_id = ?`, subscriptionID); err != nil {
		return fmt.Errorf("store: set subscription nodes: %w", err)
	}
	for _, nodeID := range nodeIDs {
		if _, err := tx.Exec(`INSERT INTO subscription_nodes (subscription_id, node_id) VALUES (?, ?)`,
			subscriptionID, nodeID); err != nil {
			return fmt.Errorf("store: set subscription nodes: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: set subscription nodes: %w", err)
	}
	return nil
}

// ListSubscriptionNodes returns the nodes a subscription grants access to,
// skipping any that are disabled.
func (s *Store) ListSubscriptionNodes(subscriptionID string) ([]Node, error) {
	rows, err := s.db.Query(`SELECT n.id, n.name, n.address, n.api_key, n.tags, n.enabled, n.last_seen_at, n.created_at
		FROM nodes n
		JOIN subscription_nodes sn ON sn.node_id = n.id
		WHERE sn.subscription_id = ? AND n.enabled = 1
		ORDER BY n.name`, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("store: list subscription nodes: %w", err)
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
		return nil, fmt.Errorf("store: list subscription nodes: %w", err)
	}
	return out, nil
}

func scanSubscription(rows *sql.Rows) (Subscription, error) {
	var sub Subscription
	var enabled int
	var createdAt string
	if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Name, &sub.Token, &enabled, &sub.ExpiresAt, &createdAt); err != nil {
		return Subscription{}, fmt.Errorf("store: scan subscription: %w", err)
	}
	sub.Enabled = enabled != 0
	var err error
	sub.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Subscription{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return sub, nil
}

func scanSubscriptionRow(row *sql.Row) (Subscription, error) {
	var sub Subscription
	var enabled int
	var createdAt string
	if err := row.Scan(&sub.ID, &sub.UserID, &sub.Name, &sub.Token, &enabled, &sub.ExpiresAt, &createdAt); err != nil {
		return Subscription{}, err
	}
	sub.Enabled = enabled != 0
	var err error
	sub.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Subscription{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return sub, nil
}
