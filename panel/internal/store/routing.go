package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RoutingRule is one entry in a RoutingRuleSet — the panel's copy of the
// same shape the client's internal/routing package defines, kept in sync by
// hand since the two live in separate Go modules with nothing shared
// between them for this. Field names/JSON tags must match exactly: this
// struct's JSON encoding is both what's stored in routing_rule_sets and
// what's handed to clients verbatim in /sub/{token}.
type RoutingRule struct {
	ID     string `json:"id"`
	Type   string `json:"type"`   // "domain" | "app" | "geoip"
	Value  string `json:"value"`
	Action string `json:"action"` // "direct" | "proxy"
}

// RoutingRuleSet is a named, admin-managed list of rules that can be
// assigned to a user (see SetUserRoutingRuleSet) and is then delivered
// alongside their node list at /sub/{token}.
type RoutingRuleSet struct {
	ID        string
	Name      string
	Rules     []RoutingRule
	CreatedAt time.Time
}

// ListRoutingRuleSets returns every rule set, newest first.
func (s *Store) ListRoutingRuleSets() ([]RoutingRuleSet, error) {
	rows, err := s.db.Query(`SELECT id, name, rules_json, created_at FROM routing_rule_sets ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list routing rule sets: %w", err)
	}
	defer rows.Close()

	var out []RoutingRuleSet
	for rows.Next() {
		rs, err := scanRoutingRuleSet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, rows.Err()
}

// GetRoutingRuleSet returns one rule set by ID.
func (s *Store) GetRoutingRuleSet(id string) (RoutingRuleSet, error) {
	row := s.db.QueryRow(`SELECT id, name, rules_json, created_at FROM routing_rule_sets WHERE id = ?`, id)
	rs, err := scanRoutingRuleSetRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingRuleSet{}, ErrNotFound
	}
	return rs, err
}

// CreateRoutingRuleSet adds a new, initially empty, named rule set.
func (s *Store) CreateRoutingRuleSet(name string) (RoutingRuleSet, error) {
	id, err := randomToken(8)
	if err != nil {
		return RoutingRuleSet{}, err
	}
	rs := RoutingRuleSet{ID: id, Name: name, Rules: []RoutingRule{}, CreatedAt: time.Now().UTC()}
	rulesJSON, err := json.Marshal(rs.Rules)
	if err != nil {
		return RoutingRuleSet{}, fmt.Errorf("store: marshal rules: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO routing_rule_sets (id, name, rules_json, created_at) VALUES (?, ?, ?, ?)`,
		rs.ID, rs.Name, string(rulesJSON), rs.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return RoutingRuleSet{}, fmt.Errorf("store: create routing rule set: %w", err)
	}
	return rs, nil
}

// UpdateRoutingRuleSet replaces a set's name and rules wholesale.
func (s *Store) UpdateRoutingRuleSet(id, name string, rules []RoutingRule) error {
	if rules == nil {
		rules = []RoutingRule{}
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("store: marshal rules: %w", err)
	}
	res, err := s.db.Exec(`UPDATE routing_rule_sets SET name = ?, rules_json = ? WHERE id = ?`, name, string(rulesJSON), id)
	if err != nil {
		return fmt.Errorf("store: update routing rule set: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update routing rule set: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRoutingRuleSet removes a rule set. Any user it was assigned to
// simply falls back to no routing rules — there's no FK to enforce
// cleanup, so callers relying on the admin UI's own confirmation dialog is
// the only guard here, same as node/release deletion elsewhere.
func (s *Store) DeleteRoutingRuleSet(id string) error {
	res, err := s.db.Exec(`DELETE FROM routing_rule_sets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete routing rule set: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete routing rule set: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	if _, err := s.db.Exec(`UPDATE users SET routing_rule_set_id = '' WHERE routing_rule_set_id = ?`, id); err != nil {
		return fmt.Errorf("store: clear deleted routing rule set from users: %w", err)
	}
	return nil
}

func scanRoutingRuleSet(rows *sql.Rows) (RoutingRuleSet, error) {
	var rs RoutingRuleSet
	var rulesJSON, createdAt string
	if err := rows.Scan(&rs.ID, &rs.Name, &rulesJSON, &createdAt); err != nil {
		return RoutingRuleSet{}, fmt.Errorf("store: scan routing rule set: %w", err)
	}
	return finishRoutingRuleSet(rs, rulesJSON, createdAt)
}

func scanRoutingRuleSetRow(row *sql.Row) (RoutingRuleSet, error) {
	var rs RoutingRuleSet
	var rulesJSON, createdAt string
	if err := row.Scan(&rs.ID, &rs.Name, &rulesJSON, &createdAt); err != nil {
		return RoutingRuleSet{}, err
	}
	return finishRoutingRuleSet(rs, rulesJSON, createdAt)
}

func finishRoutingRuleSet(rs RoutingRuleSet, rulesJSON, createdAt string) (RoutingRuleSet, error) {
	if err := json.Unmarshal([]byte(rulesJSON), &rs.Rules); err != nil {
		return RoutingRuleSet{}, fmt.Errorf("store: parse rules_json: %w", err)
	}
	var err error
	rs.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return RoutingRuleSet{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return rs, nil
}
