package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rookery/node/internal/store"
	"github.com/rookery/shared/signaling"
)

func newTestServer(t *testing.T) (*Server, store.User) {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "rookery.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { st.Close() })

	u, err := st.CreateUser("test-user")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	return &Server{store: st}, u
}

func TestAuthenticateAcceptsValidSignature(t *testing.T) {
	s, u := newTestServer(t)
	body := []byte(`{"user_id":"` + u.ID + `","sdp":"v=0..."}`)
	now := time.Now()
	sig := signaling.Sign([]byte(u.Secret), now.Unix(), body)

	r := httptest.NewRequest(http.MethodPost, "/session", nil)
	r.Header.Set(signaling.HeaderTimestamp, strconv.FormatInt(now.Unix(), 10))
	r.Header.Set(signaling.HeaderSignature, sig)

	if err := s.authenticate(r, u.ID, body); err != nil {
		t.Fatalf("authenticate() error = %v, want nil", err)
	}
}

func TestAuthenticateRejectsInvalidRequests(t *testing.T) {
	s, u := newTestServer(t)
	body := []byte(`{"user_id":"` + u.ID + `","sdp":"v=0..."}`)
	now := time.Now()
	validSig := signaling.Sign([]byte(u.Secret), now.Unix(), body)

	cases := []struct {
		name      string
		userID    string
		timestamp string
		signature string
	}{
		{"missing-headers", u.ID, "", ""},
		{"missing-signature", u.ID, strconv.FormatInt(now.Unix(), 10), ""},
		{"missing-timestamp", u.ID, "", validSig},
		{"unknown-user", "no-such-user", strconv.FormatInt(now.Unix(), 10), validSig},
		{"empty-user", "", strconv.FormatInt(now.Unix(), 10), validSig},
		{"wrong-secret-signature", u.ID, strconv.FormatInt(now.Unix(), 10), signaling.Sign([]byte("other-secret"), now.Unix(), body)},
		{"expired-timestamp", u.ID, strconv.FormatInt(now.Add(-time.Hour).Unix(), 10), signaling.Sign([]byte(u.Secret), now.Add(-time.Hour).Unix(), body)},
		{"garbage-signature", u.ID, strconv.FormatInt(now.Unix(), 10), "not-hex!!"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/session", nil)
			r.Header.Set(signaling.HeaderTimestamp, tc.timestamp)
			r.Header.Set(signaling.HeaderSignature, tc.signature)

			if err := s.authenticate(r, tc.userID, body); err == nil {
				t.Fatalf("authenticate() expected error, got nil")
			}
		})
	}
}

func TestHandleSessionReturnsNotFoundForUnauthenticatedRequest(t *testing.T) {
	s, _ := newTestServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /session", s.handleSession)

	body := []byte(`{"user_id":"nobody","sdp":"v=0..."}`)
	r := httptest.NewRequest(http.MethodPost, "/session", bytes.NewReader(body))
	r.Header.Set(signaling.HeaderTimestamp, "not-a-timestamp")
	r.Header.Set(signaling.HeaderSignature, "not-a-signature")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
