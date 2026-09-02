package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "panel.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateListGetDeleteUser(t *testing.T) {
	s := openTestStore(t)

	u1, err := s.CreateUser("alice")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	u2, err := s.CreateUser("bob")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u1.ID == u2.ID || u1.Token == u2.Token {
		t.Fatalf("CreateUser() produced colliding ID/token")
	}
	if !u1.Enabled {
		t.Fatalf("CreateUser() should default to enabled")
	}

	got, err := s.GetUser(u1.ID)
	if err != nil || got != u1 {
		t.Fatalf("GetUser(%q) = %+v, %v; want %+v, nil", u1.ID, got, err, u1)
	}

	byToken, err := s.GetUserByToken(u1.Token)
	if err != nil || byToken != u1 {
		t.Fatalf("GetUserByToken() = %+v, %v; want %+v, nil", byToken, err, u1)
	}

	list, err := s.ListUsers()
	if err != nil || len(list) != 2 {
		t.Fatalf("ListUsers() = %v, %v; want 2 users, nil", list, err)
	}

	if err := s.DeleteUser(u1.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if _, err := s.GetUser(u1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetUser() after delete error = %v, want ErrNotFound", err)
	}
}

func TestUpdateUser(t *testing.T) {
	s := openTestStore(t)
	u, err := s.CreateUser("alice")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if err := s.UpdateUser(u.ID, "alice renamed", false, "2026-01-01T00:00:00Z", "2027-01-01T00:00:00Z"); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	got, err := s.GetUser(u.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.Name != "alice renamed" || got.Enabled || got.StartsAt != "2026-01-01T00:00:00Z" || got.ExpiresAt != "2027-01-01T00:00:00Z" {
		t.Fatalf("GetUser() after update = %+v, want updated name/enabled/window", got)
	}
}

func TestCreateListDeleteNode(t *testing.T) {
	s := openTestStore(t)

	n, err := s.CreateNode("de-1", "https://de1.example.com", "de,cheap")
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	if n.APIKey == "" {
		t.Fatalf("CreateNode() returned empty APIKey")
	}

	if _, err := s.NodeByAPIKey(n.ID, n.APIKey); err != nil {
		t.Fatalf("NodeByAPIKey() with correct key error = %v", err)
	}
	if _, err := s.NodeByAPIKey(n.ID, "wrong-key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("NodeByAPIKey() with wrong key error = %v, want ErrNotFound", err)
	}

	if err := s.TouchNode(n.ID); err != nil {
		t.Fatalf("TouchNode() error = %v", err)
	}
	got, err := s.GetNode(n.ID)
	if err != nil || got.LastSeenAt == "" {
		t.Fatalf("GetNode() after TouchNode: LastSeenAt still empty, err = %v", err)
	}

	if err := s.DeleteNode(n.ID); err != nil {
		t.Fatalf("DeleteNode() error = %v", err)
	}
	if _, err := s.GetNode(n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNode() after delete error = %v, want ErrNotFound", err)
	}
}

func TestUserNodeAssignmentAndLookupByToken(t *testing.T) {
	s := openTestStore(t)

	u, err := s.CreateUser("alice")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	n1, err := s.CreateNode("de-1", "https://de1.example.com", "")
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	n2, err := s.CreateNode("nl-1", "https://nl1.example.com", "")
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	if err := s.SetUserNodes(u.ID, []string{n1.ID, n2.ID}); err != nil {
		t.Fatalf("SetUserNodes() error = %v", err)
	}
	nodes, err := s.ListUserNodes(u.ID)
	if err != nil || len(nodes) != 2 {
		t.Fatalf("ListUserNodes() = %v, %v; want 2 nodes, nil", nodes, err)
	}

	// Replacing the set drops what's no longer included.
	if err := s.SetUserNodes(u.ID, []string{n1.ID}); err != nil {
		t.Fatalf("SetUserNodes() error = %v", err)
	}
	nodes, err = s.ListUserNodes(u.ID)
	if err != nil || len(nodes) != 1 || nodes[0].ID != n1.ID {
		t.Fatalf("ListUserNodes() after replace = %v, %v; want just %q", nodes, err, n1.ID)
	}

	byToken, err := s.GetUserByToken(u.Token)
	if err != nil || byToken.ID != u.ID {
		t.Fatalf("GetUserByToken() = %+v, %v; want ID %q", byToken, err, u.ID)
	}

	if err := s.DeleteNode(n1.ID); err != nil {
		t.Fatalf("DeleteNode() error = %v", err)
	}
	nodes, err = s.ListUserNodes(u.ID)
	if err != nil || len(nodes) != 0 {
		t.Fatalf("ListUserNodes() after node deleted = %v, %v; want empty (cascade)", nodes, err)
	}
}

func TestRecordTrafficAndTotals(t *testing.T) {
	s := openTestStore(t)

	u, err := s.CreateUser("alice")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	n, err := s.CreateNode("de-1", "https://de1.example.com", "")
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	if err := s.RecordTraffic(u.ID, n.ID, 100, 200); err != nil {
		t.Fatalf("RecordTraffic() error = %v", err)
	}
	if err := s.RecordTraffic(u.ID, n.ID, 50, 25); err != nil {
		t.Fatalf("RecordTraffic() error = %v", err)
	}

	userTotals, err := s.TotalsForUser(u.ID)
	if err != nil || userTotals.BytesUp != 150 || userTotals.BytesDown != 225 {
		t.Fatalf("TotalsForUser() = %+v, %v; want {150 225}, nil", userTotals, err)
	}

	nodeTotals, err := s.TotalsForNode(n.ID)
	if err != nil || nodeTotals != userTotals {
		t.Fatalf("TotalsForNode() = %+v, %v; want %+v, nil", nodeTotals, err, userTotals)
	}

	global, err := s.GlobalTotals()
	if err != nil || global != userTotals {
		t.Fatalf("GlobalTotals() = %+v, %v; want %+v, nil", global, err, userTotals)
	}

	series, err := s.GlobalTimeSeries(24)
	if err != nil {
		t.Fatalf("GlobalTimeSeries() error = %v", err)
	}
	if len(series) == 0 {
		t.Fatalf("GlobalTimeSeries() returned no points")
	}
	var seriesUp uint64
	for _, p := range series {
		seriesUp += p.BytesUp
	}
	if seriesUp != userTotals.BytesUp {
		t.Fatalf("GlobalTimeSeries() sums to %d up, want %d", seriesUp, userTotals.BytesUp)
	}
}

func TestEnsureAdminGeneratesOnce(t *testing.T) {
	s := openTestStore(t)

	password, err := s.EnsureAdmin()
	if err != nil || password == "" {
		t.Fatalf("EnsureAdmin() = %q, %v; want non-empty, nil", password, err)
	}
	if !s.VerifyAdmin(defaultAdminUsername, password) {
		t.Fatalf("VerifyAdmin() failed with just-generated password")
	}

	second, err := s.EnsureAdmin()
	if err != nil || second != "" {
		t.Fatalf("second EnsureAdmin() = %q, %v; want empty, nil", second, err)
	}
}

func TestPublicAddr(t *testing.T) {
	s := openTestStore(t)

	addr, err := s.PublicAddr()
	if err != nil || addr != "" {
		t.Fatalf("PublicAddr() = %q, %v; want empty, nil", addr, err)
	}

	if err := s.SetPublicAddr("https://panel.example.com"); err != nil {
		t.Fatalf("SetPublicAddr() error = %v", err)
	}
	addr, err = s.PublicAddr()
	if err != nil || addr != "https://panel.example.com" {
		t.Fatalf("PublicAddr() = %q, %v; want https://panel.example.com, nil", addr, err)
	}
}
