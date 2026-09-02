package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Release is one uploaded client build.
type Release struct {
	ID        string
	Version   string
	Notes     string
	Filename  string
	FilePath  string // absolute path on disk
	Size      int64
	CreatedAt time.Time
}

// ListReleases returns every release, newest first.
func (s *Store) ListReleases() ([]Release, error) {
	rows, err := s.db.Query(`SELECT id, version, notes, filename, file_path, size, created_at
		FROM releases ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list releases: %w", err)
	}
	defer rows.Close()

	var out []Release
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LatestRelease returns the most recently uploaded release.
func (s *Store) LatestRelease() (Release, error) {
	row := s.db.QueryRow(`SELECT id, version, notes, filename, file_path, size, created_at
		FROM releases ORDER BY created_at DESC LIMIT 1`)
	r, err := scanReleaseRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Release{}, ErrNotFound
	}
	return r, err
}

// GetRelease returns one release by ID.
func (s *Store) GetRelease(id string) (Release, error) {
	row := s.db.QueryRow(`SELECT id, version, notes, filename, file_path, size, created_at
		FROM releases WHERE id = ?`, id)
	r, err := scanReleaseRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Release{}, ErrNotFound
	}
	return r, err
}

// CreateRelease records a newly uploaded release. id and filePath are the
// caller's — the caller already had to pick id to name the file's directory
// on disk before it could save the upload, so CreateRelease just persists
// that choice rather than generating its own.
func (s *Store) CreateRelease(id, version, notes, filename, filePath string, size int64) (Release, error) {
	r := Release{ID: id, Version: version, Notes: notes, Filename: filename, FilePath: filePath, Size: size, CreatedAt: time.Now().UTC()}
	_, err := s.db.Exec(`INSERT INTO releases (id, version, notes, filename, file_path, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Version, r.Notes, r.Filename, r.FilePath, r.Size, r.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Release{}, fmt.Errorf("store: create release: %w", err)
	}
	return r, nil
}

// DeleteRelease removes a release's DB row. The caller is responsible for
// removing the file at its FilePath.
func (s *Store) DeleteRelease(id string) error {
	res, err := s.db.Exec(`DELETE FROM releases WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete release: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete release: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanRelease(rows *sql.Rows) (Release, error) {
	var r Release
	var createdAt string
	if err := rows.Scan(&r.ID, &r.Version, &r.Notes, &r.Filename, &r.FilePath, &r.Size, &createdAt); err != nil {
		return Release{}, fmt.Errorf("store: scan release: %w", err)
	}
	var err error
	r.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Release{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return r, nil
}

func scanReleaseRow(row *sql.Row) (Release, error) {
	var r Release
	var createdAt string
	if err := row.Scan(&r.ID, &r.Version, &r.Notes, &r.Filename, &r.FilePath, &r.Size, &createdAt); err != nil {
		return Release{}, err
	}
	var err error
	r.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Release{}, fmt.Errorf("store: parse created_at: %w", err)
	}
	return r, nil
}
