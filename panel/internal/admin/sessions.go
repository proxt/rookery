package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// sessionTTL is how long an admin login stays valid.
const sessionTTL = 24 * time.Hour

type sessionEntry struct {
	adminID string
	expiry  time.Time
}

// sessionStore is an in-memory set of valid session tokens, each tied to
// the admin that logged in with it. Sessions are lost on restart, which
// just forces a re-login.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]sessionEntry // token -> entry
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]sessionEntry)}
}

func (s *sessionStore) create(adminID string) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[token] = sessionEntry{adminID: adminID, expiry: time.Now().Add(sessionTTL)}
	s.mu.Unlock()

	return token, nil
}

// get returns the admin ID a valid, unexpired token belongs to.
func (s *sessionStore) get(token string) (adminID string, ok bool) {
	if token == "" {
		return "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.sessions[token]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiry) {
		delete(s.sessions, token)
		return "", false
	}
	return entry.adminID, true
}

func (s *sessionStore) revoke(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("admin: generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
