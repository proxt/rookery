package signaling

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	secret := []byte("shared-secret")
	now := time.Unix(1_700_000_000, 0)
	body := []byte(`{"sdp":"v=0..."}`)

	sig := Sign(secret, now.Unix(), body)
	if err := Verify(secret, strconv.FormatInt(now.Unix(), 10), body, sig, now); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}
}

func TestVerifyWithinClockSkewWindow(t *testing.T) {
	secret := []byte("shared-secret")
	signedAt := time.Unix(1_700_000_000, 0)
	body := []byte("payload")
	sig := Sign(secret, signedAt.Unix(), body)

	cases := []struct {
		name    string
		now     time.Time
		wantErr bool
	}{
		{"exact-time", signedAt, false},
		{"59s-later", signedAt.Add(59 * time.Second), false},
		{"60s-later-boundary", signedAt.Add(60 * time.Second), false},
		{"61s-later-too-late", signedAt.Add(61 * time.Second), true},
		{"59s-earlier", signedAt.Add(-59 * time.Second), false},
		{"61s-earlier-too-early", signedAt.Add(-61 * time.Second), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify(secret, strconv.FormatInt(signedAt.Unix(), 10), body, sig, tc.now)
			if tc.wantErr && !errors.Is(err, ErrClockSkew) {
				t.Fatalf("Verify() error = %v, want ErrClockSkew", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Verify() error = %v, want nil", err)
			}
		})
	}
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	secret := []byte("shared-secret")
	now := time.Unix(1_700_000_000, 0)
	sig := Sign(secret, now.Unix(), []byte("original-body"))

	err := Verify(secret, strconv.FormatInt(now.Unix(), 10), []byte("tampered-body"), sig, now)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify() error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	body := []byte("payload")
	sig := Sign([]byte("secret-a"), now.Unix(), body)

	err := Verify([]byte("secret-b"), strconv.FormatInt(now.Unix(), 10), body, sig, now)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("Verify() error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyGarbageInput(t *testing.T) {
	secret := []byte("shared-secret")
	now := time.Unix(1_700_000_000, 0)
	body := []byte("payload")
	validSig := Sign(secret, now.Unix(), body)

	cases := []struct {
		name      string
		timestamp string
		signature string
		wantErr   error
	}{
		{"empty-timestamp", "", validSig, ErrMissingTimestamp},
		{"non-numeric-timestamp", "not-a-number", validSig, ErrInvalidTimestamp},
		{"negative-timestamp", "-1", validSig, nil},
		{"empty-signature", strconv.FormatInt(now.Unix(), 10), "", ErrInvalidSignature},
		{"non-hex-signature", strconv.FormatInt(now.Unix(), 10), "not-valid-hex!!", ErrInvalidSignature},
		{"short-signature", strconv.FormatInt(now.Unix(), 10), "ab", ErrInvalidSignature},
		{"truncated-valid-hex", strconv.FormatInt(now.Unix(), 10), validSig[:len(validSig)-2], ErrInvalidSignature},
		{"timestamp-with-decimal", strconv.FormatInt(now.Unix(), 10) + ".5", validSig, ErrInvalidTimestamp},
		{"timestamp-overflow", "99999999999999999999999999", validSig, ErrInvalidTimestamp},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Verify(secret, tc.timestamp, body, tc.signature, now)
			if err == nil {
				t.Fatalf("Verify() expected error, got nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Verify() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSignDeterministic(t *testing.T) {
	secret := []byte("shared-secret")
	body := []byte("payload")
	sig1 := Sign(secret, 1700000000, body)
	sig2 := Sign(secret, 1700000000, body)
	if sig1 != sig2 {
		t.Fatalf("Sign() not deterministic: %q != %q", sig1, sig2)
	}
}

func TestSignDiffersOnTimestampOrBody(t *testing.T) {
	secret := []byte("shared-secret")
	base := Sign(secret, 1700000000, []byte("payload"))

	if diffTs := Sign(secret, 1700000001, []byte("payload")); diffTs == base {
		t.Fatalf("Sign() unchanged when timestamp changed")
	}
	if diffBody := Sign(secret, 1700000000, []byte("other-payload")); diffBody == base {
		t.Fatalf("Sign() unchanged when body changed")
	}
}
