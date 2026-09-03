// Package store is the panel's SQLite-backed persistence: users,
// subscriptions, the nodes they grant access to, admin login, and traffic
// stats reported by nodes.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Store wraps the panel's single SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and runs
// migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// modernc.org/sqlite serializes access to a single file at the driver
	// level; capping the pool at one connection avoids SQLITE_BUSY churn
	// from concurrent writers without needing WAL/busy-timeout tuning. It
	// also means this pragma only needs setting once, not per-connection.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		-- Multiple admins can log in independently — no more singleton row.
		CREATE TABLE IF NOT EXISTS admins (
			id            TEXT PRIMARY KEY,
			username      TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at    TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS settings (
			id                  INTEGER PRIMARY KEY CHECK (id = 1),
			public_addr         TEXT NOT NULL DEFAULT '',
			auto_update_enabled INTEGER NOT NULL DEFAULT 1
		);
		INSERT OR IGNORE INTO settings (id, public_addr) VALUES (1, '');

		-- A user IS a subscription: one token, one link, one set of nodes.
		-- No separate subscription entity — matches how Marzban/Remnawave/
		-- happ model this (one subscription URL per user), rather than a
		-- user owning multiple named subscriptions.
		CREATE TABLE IF NOT EXISTS users (
			id             TEXT PRIMARY KEY,
			name           TEXT NOT NULL,
			token          TEXT NOT NULL UNIQUE,
			enabled        INTEGER NOT NULL DEFAULT 1,
			starts_at      TEXT NOT NULL DEFAULT '',
			expires_at     TEXT NOT NULL DEFAULT '',
			last_active_at TEXT NOT NULL DEFAULT '',
			created_at     TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS nodes (
			id            TEXT PRIMARY KEY,
			name          TEXT NOT NULL,
			address       TEXT NOT NULL,
			api_key       TEXT NOT NULL,
			tags          TEXT NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			last_seen_at  TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS user_nodes (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			PRIMARY KEY (user_id, node_id)
		);

		CREATE TABLE IF NOT EXISTS stat_samples (
			user_id     TEXT NOT NULL,
			node_id     TEXT NOT NULL,
			bucket_hour TEXT NOT NULL,
			bytes_up    INTEGER NOT NULL DEFAULT 0,
			bytes_down  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (user_id, node_id, bucket_hour)
		);
		CREATE INDEX IF NOT EXISTS idx_stat_samples_bucket ON stat_samples(bucket_hour);

		CREATE TABLE IF NOT EXISTS releases (
			id         TEXT PRIMARY KEY,
			version    TEXT NOT NULL,
			notes      TEXT NOT NULL DEFAULT '',
			filename   TEXT NOT NULL,
			file_path  TEXT NOT NULL,
			size       INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);

		-- admin_name is denormalized (not a FK to admins) so a log entry stays
		-- readable after the admin who made it is deleted.
		CREATE TABLE IF NOT EXISTS audit_log (
			id          TEXT PRIMARY KEY,
			admin_id    TEXT NOT NULL,
			admin_name  TEXT NOT NULL,
			action      TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT '',
			target_id   TEXT NOT NULL DEFAULT '',
			detail      TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at);
	`)
	if err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}

	// CREATE TABLE IF NOT EXISTS above doesn't add columns to a table that
	// already exists from an earlier version — do that by hand, tolerating
	// the "already there" case, so upgrading doesn't require wiping data.
	if err := s.addColumnIfMissing("users", "last_active_at", `TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("settings", "auto_update_enabled", `INTEGER NOT NULL DEFAULT 1`); err != nil {
		return err
	}
	return nil
}

func (s *Store) addColumnIfMissing(table, column, def string) error {
	_, err := s.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, def))
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("store: add column %s.%s: %w", table, column, err)
	}
	return nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
