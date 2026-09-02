package store

import (
	"fmt"
	"time"
)

// bucketLayout truncates a timestamp to the hour for stat_samples' bucket
// key. Lexicographic string ordering matches chronological ordering, which
// range queries (GlobalTotalsSince, TimeSeries) rely on.
const bucketLayout = "2006-01-02T15"

// Totals is a bytes-up/bytes-down traffic sum.
type Totals struct {
	BytesUp   uint64
	BytesDown uint64
}

// TimeSeriesPoint is one hourly bucket's traffic, for charting.
type TimeSeriesPoint struct {
	BucketHour string // "2006-01-02T15"
	BytesUp    uint64
	BytesDown  uint64
}

// RecordTraffic adds a reported traffic delta for one user/node pair into
// the current UTC-hour bucket.
func (s *Store) RecordTraffic(userID, nodeID string, bytesUp, bytesDown uint64) error {
	bucket := time.Now().UTC().Format(bucketLayout)
	_, err := s.db.Exec(`
		INSERT INTO stat_samples (user_id, node_id, bucket_hour, bytes_up, bytes_down)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, node_id, bucket_hour)
		DO UPDATE SET bytes_up = bytes_up + excluded.bytes_up, bytes_down = bytes_down + excluded.bytes_down
	`, userID, nodeID, bucket, bytesUp, bytesDown)
	if err != nil {
		return fmt.Errorf("store: record traffic: %w", err)
	}
	return nil
}

// TotalsForUser sums all-time traffic for one user.
func (s *Store) TotalsForUser(userID string) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE user_id = ?`, userID)
}

// TotalsForNode sums all-time traffic relayed by one node.
func (s *Store) TotalsForNode(nodeID string) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE node_id = ?`, nodeID)
}

// GlobalTotals sums all-time traffic across every user and node.
func (s *Store) GlobalTotals() (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0) FROM stat_samples`)
}

// GlobalTotalsSince sums traffic recorded at or after since.
func (s *Store) GlobalTotalsSince(since time.Time) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE bucket_hour >= ?`, since.UTC().Format(bucketLayout))
}

// GlobalTimeSeries returns one point per hour for the last hours hours
// (oldest first), summed across every user and node — for the dashboard
// traffic chart. Hours with no traffic are included as zero points so the
// chart has a continuous x-axis.
func (s *Store) GlobalTimeSeries(hours int) ([]TimeSeriesPoint, error) {
	return s.timeSeries(`SELECT bucket_hour, COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE bucket_hour >= ? GROUP BY bucket_hour`, hours)
}

// UserTimeSeries is GlobalTimeSeries scoped to one user.
func (s *Store) UserTimeSeries(userID string, hours int) ([]TimeSeriesPoint, error) {
	return s.timeSeries(`SELECT bucket_hour, COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE user_id = ? AND bucket_hour >= ? GROUP BY bucket_hour`, hours, userID)
}

func (s *Store) timeSeries(query string, hours int, extraArgs ...any) ([]TimeSeriesPoint, error) {
	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	args := append(append([]any{}, extraArgs...), since.Format(bucketLayout))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: time series: %w", err)
	}
	defer rows.Close()

	byBucket := make(map[string]TimeSeriesPoint)
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.BucketHour, &p.BytesUp, &p.BytesDown); err != nil {
			return nil, fmt.Errorf("store: scan time series point: %w", err)
		}
		byBucket[p.BucketHour] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: time series: %w", err)
	}

	out := make([]TimeSeriesPoint, 0, hours+1)
	for t := since; !t.After(now); t = t.Add(time.Hour) {
		bucket := t.Format(bucketLayout)
		if p, ok := byBucket[bucket]; ok {
			out = append(out, p)
		} else {
			out = append(out, TimeSeriesPoint{BucketHour: bucket})
		}
	}
	return out, nil
}

func (s *Store) queryTotals(query string, args ...any) (Totals, error) {
	var t Totals
	if err := s.db.QueryRow(query, args...).Scan(&t.BytesUp, &t.BytesDown); err != nil {
		return Totals{}, fmt.Errorf("store: totals: %w", err)
	}
	return t, nil
}
