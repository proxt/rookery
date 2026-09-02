package store

import (
	"fmt"
	"time"
)

// bucketLayout truncates a timestamp to the hour for stat_samples' bucket
// key. Lexicographic string ordering matches chronological ordering, which
// range queries (TotalsSince) rely on.
const bucketLayout = "2006-01-02T15"

// Totals is a bytes-up/bytes-down traffic sum.
type Totals struct {
	BytesUp   uint64
	BytesDown uint64
}

// RecordTraffic adds a reported traffic delta for one subscription/node
// pair into the current UTC-hour bucket.
func (s *Store) RecordTraffic(subscriptionID, nodeID string, bytesUp, bytesDown uint64) error {
	bucket := time.Now().UTC().Format(bucketLayout)
	_, err := s.db.Exec(`
		INSERT INTO stat_samples (subscription_id, node_id, bucket_hour, bytes_up, bytes_down)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(subscription_id, node_id, bucket_hour)
		DO UPDATE SET bytes_up = bytes_up + excluded.bytes_up, bytes_down = bytes_down + excluded.bytes_down
	`, subscriptionID, nodeID, bucket, bytesUp, bytesDown)
	if err != nil {
		return fmt.Errorf("store: record traffic: %w", err)
	}
	return nil
}

// TotalsForSubscription sums all-time traffic for one subscription.
func (s *Store) TotalsForSubscription(subscriptionID string) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE subscription_id = ?`, subscriptionID)
}

// TotalsForUser sums all-time traffic across every subscription a user owns.
func (s *Store) TotalsForUser(userID string) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(ss.bytes_up),0), COALESCE(SUM(ss.bytes_down),0)
		FROM stat_samples ss JOIN subscriptions sub ON sub.id = ss.subscription_id
		WHERE sub.user_id = ?`, userID)
}

// TotalsForNode sums all-time traffic relayed by one node.
func (s *Store) TotalsForNode(nodeID string) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE node_id = ?`, nodeID)
}

// GlobalTotals sums all-time traffic across every subscription and node.
func (s *Store) GlobalTotals() (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0) FROM stat_samples`)
}

// GlobalTotalsSince sums traffic recorded at or after since.
func (s *Store) GlobalTotalsSince(since time.Time) (Totals, error) {
	return s.queryTotals(`SELECT COALESCE(SUM(bytes_up),0), COALESCE(SUM(bytes_down),0)
		FROM stat_samples WHERE bucket_hour >= ?`, since.UTC().Format(bucketLayout))
}

func (s *Store) queryTotals(query string, args ...any) (Totals, error) {
	var t Totals
	if err := s.db.QueryRow(query, args...).Scan(&t.BytesUp, &t.BytesDown); err != nil {
		return Totals{}, fmt.Errorf("store: totals: %w", err)
	}
	return t, nil
}
