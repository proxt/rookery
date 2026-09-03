package store

import (
	"database/sql"
	"fmt"
	"time"
)

// AuditEntry is one recorded admin action.
type AuditEntry struct {
	ID         string
	AdminID    string
	AdminName  string
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	CreatedAt  time.Time
}

// CreateAuditEntry records one admin action. Append-only — there is no
// update/delete, and callers are expected to treat failures as best-effort
// (log and continue) rather than fail the action being audited.
func (s *Store) CreateAuditEntry(adminID, adminName, action, targetType, targetID, detail string) error {
	id, err := randomToken(8)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO audit_log (id, admin_id, admin_name, action, target_type, target_id, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, adminID, adminName, action, targetType, targetID, detail, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: create audit entry: %w", err)
	}
	return nil
}

// PruneAuditLog deletes entries older than olderThan and returns how many
// rows were removed. Called periodically (see server.auditLogRetention) so
// the table doesn't grow forever — audit history is a rolling window, not
// permanent storage.
func (s *Store) PruneAuditLog(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339Nano)
	res, err := s.db.Exec(`DELETE FROM audit_log WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("store: prune audit log: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: prune audit log: %w", err)
	}
	return n, nil
}

// ListAuditLog returns audit entries newest-first, limit capped by the
// caller (see admin.handleListAuditLog).
func (s *Store) ListAuditLog(limit, offset int) ([]AuditEntry, error) {
	rows, err := s.db.Query(`SELECT id, admin_id, admin_name, action, target_type, target_id, detail, created_at
		FROM audit_log ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list audit log: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		e, err := scanAuditEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanAuditEntry(rows *sql.Rows) (AuditEntry, error) {
	var e AuditEntry
	var createdAt string
	if err := rows.Scan(&e.ID, &e.AdminID, &e.AdminName, &e.Action, &e.TargetType, &e.TargetID, &e.Detail, &createdAt); err != nil {
		return AuditEntry{}, fmt.Errorf("store: scan audit entry: %w", err)
	}
	var err error
	e.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return AuditEntry{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return e, nil
}
