package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "rookery.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateListGetDeleteUser(t *testing.T) {
	s := openTestStore(t)

	if list, err := s.ListUsers(); err != nil || len(list) != 0 {
		t.Fatalf("ListUsers() = %v, %v; want empty, nil", list, err)
	}

	u1, err := s.CreateUser("laptop")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u1.ID == "" || u1.Secret == "" {
		t.Fatalf("CreateUser() returned user with empty ID/Secret: %+v", u1)
	}

	u2, err := s.CreateUser("phone")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u1.ID == u2.ID || u1.Secret == u2.Secret {
		t.Fatalf("CreateUser() produced colliding ID/Secret: %+v vs %+v", u1, u2)
	}

	got, err := s.GetUser(u1.ID)
	if err != nil || got != u1 {
		t.Fatalf("GetUser(%q) = %+v, %v; want %+v, nil", u1.ID, got, err, u1)
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
	if err := s.DeleteUser(u1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteUser() on already-deleted user error = %v, want ErrNotFound", err)
	}
}

func TestPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rookery.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	u, err := s1.CreateUser("laptop")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := s1.EnsureAdmin(); err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	if err := s1.SetPublicAddr("https://vpn.example.com"); err != nil {
		t.Fatalf("SetPublicAddr() error = %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	defer s2.Close()

	got, err := s2.GetUser(u.ID)
	if err != nil || got != u {
		t.Fatalf("reloaded user = %+v, %v; want %+v, nil", got, err, u)
	}

	addr, err := s2.PublicAddr()
	if err != nil || addr != "https://vpn.example.com" {
		t.Fatalf("PublicAddr() = %q, %v; want https://vpn.example.com, nil", addr, err)
	}
}

func TestEnsureAdminGeneratesOnce(t *testing.T) {
	s := openTestStore(t)

	password, err := s.EnsureAdmin()
	if err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}
	if password == "" {
		t.Fatalf("EnsureAdmin() generated empty password")
	}
	if !s.VerifyAdmin(defaultAdminUsername, password) {
		t.Fatalf("VerifyAdmin() failed with just-generated password")
	}

	second, err := s.EnsureAdmin()
	if err != nil {
		t.Fatalf("second EnsureAdmin() error = %v", err)
	}
	if second != "" {
		t.Fatalf("EnsureAdmin() regenerated password on second call: %q", second)
	}
	if !s.VerifyAdmin(defaultAdminUsername, password) {
		t.Fatalf("VerifyAdmin() failed after second EnsureAdmin() call")
	}
}

func TestVerifyAdminRejectsWrongCredentials(t *testing.T) {
	s := openTestStore(t)
	password, err := s.EnsureAdmin()
	if err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}

	if s.VerifyAdmin(defaultAdminUsername, "wrong-password") {
		t.Fatalf("VerifyAdmin() succeeded with wrong password")
	}
	if s.VerifyAdmin("wrong-username", password) {
		t.Fatalf("VerifyAdmin() succeeded with wrong username")
	}
}

func TestUpdateAdmin(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.EnsureAdmin(); err != nil {
		t.Fatalf("EnsureAdmin() error = %v", err)
	}

	if err := s.UpdateAdmin("newadmin", "newpassword123"); err != nil {
		t.Fatalf("UpdateAdmin() error = %v", err)
	}

	if !s.VerifyAdmin("newadmin", "newpassword123") {
		t.Fatalf("VerifyAdmin() failed with updated credentials")
	}

	username, err := s.AdminUsername()
	if err != nil || username != "newadmin" {
		t.Fatalf("AdminUsername() = %q, %v; want newadmin, nil", username, err)
	}

	// Updating only the username should leave the password intact.
	if err := s.UpdateAdmin("anotheradmin", ""); err != nil {
		t.Fatalf("UpdateAdmin() error = %v", err)
	}
	if !s.VerifyAdmin("anotheradmin", "newpassword123") {
		t.Fatalf("VerifyAdmin() failed after username-only update")
	}
}

func TestPublicAddrDefaultsEmpty(t *testing.T) {
	s := openTestStore(t)
	addr, err := s.PublicAddr()
	if err != nil || addr != "" {
		t.Fatalf("PublicAddr() = %q, %v; want empty, nil", addr, err)
	}
}

func TestSetPublicAddrIfEmpty(t *testing.T) {
	s := openTestStore(t)

	changed, err := s.SetPublicAddrIfEmpty("https://auto-detected.example.com")
	if err != nil || !changed {
		t.Fatalf("SetPublicAddrIfEmpty() = %v, %v; want true, nil", changed, err)
	}

	changed, err = s.SetPublicAddrIfEmpty("https://should-not-apply.example.com")
	if err != nil || changed {
		t.Fatalf("SetPublicAddrIfEmpty() on non-empty = %v, %v; want false, nil", changed, err)
	}

	addr, err := s.PublicAddr()
	if err != nil || addr != "https://auto-detected.example.com" {
		t.Fatalf("PublicAddr() = %q, %v; want https://auto-detected.example.com, nil", addr, err)
	}
}
