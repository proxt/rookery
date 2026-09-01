package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/rookery/shared/signaling"
)

func newTestServer(secret string) *Server {
	return &Server{secret: []byte(secret)}
}

func TestAuthenticateAcceptsValidSignature(t *testing.T) {
	s := newTestServer("shared-secret")
	body := []byte(`{"sdp":"v=0..."}`)
	now := time.Now()
	sig := signaling.Sign(s.secret, now.Unix(), body)

	r := httptest.NewRequest(http.MethodPost, "/session", nil)
	r.Header.Set(signaling.HeaderTimestamp, strconv.FormatInt(now.Unix(), 10))
	r.Header.Set(signaling.HeaderSignature, sig)

	if err := s.authenticate(r, body); err != nil {
		t.Fatalf("authenticate() error = %v, want nil", err)
	}
}

func TestAuthenticateRejectsInvalidRequests(t *testing.T) {
	s := newTestServer("shared-secret")
	body := []byte(`{"sdp":"v=0..."}`)
	now := time.Now()
	validSig := signaling.Sign(s.secret, now.Unix(), body)

	cases := []struct {
		name      string
		timestamp string
		signature string
	}{
		{"missing-headers", "", ""},
		{"missing-signature", strconv.FormatInt(now.Unix(), 10), ""},
		{"missing-timestamp", "", validSig},
		{"wrong-secret-signature", strconv.FormatInt(now.Unix(), 10), signaling.Sign([]byte("other-secret"), now.Unix(), body)},
		{"expired-timestamp", strconv.FormatInt(now.Add(-time.Hour).Unix(), 10), signaling.Sign(s.secret, now.Add(-time.Hour).Unix(), body)},
		{"garbage-signature", strconv.FormatInt(now.Unix(), 10), "not-hex!!"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/session", nil)
			r.Header.Set(signaling.HeaderTimestamp, tc.timestamp)
			r.Header.Set(signaling.HeaderSignature, tc.signature)

			if err := s.authenticate(r, body); err == nil {
				t.Fatalf("authenticate() expected error, got nil")
			}
		})
	}
}

func TestHandleSessionReturnsNotFoundForUnauthenticatedRequest(t *testing.T) {
	s := newTestServer("shared-secret")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /session", s.handleSession)

	body := []byte(`{"sdp":"v=0..."}`)
	r := httptest.NewRequest(http.MethodPost, "/session", bytes.NewReader(body))
	r.Header.Set(signaling.HeaderTimestamp, "not-a-timestamp")
	r.Header.Set(signaling.HeaderSignature, "not-a-signature")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
