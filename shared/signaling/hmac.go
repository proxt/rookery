package signaling

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	// HeaderSignature carries the hex-encoded HMAC-SHA256 of the signed message.
	HeaderSignature = "X-Signature"
	// HeaderTimestamp carries the Unix timestamp (seconds) the request was signed at.
	HeaderTimestamp = "X-Timestamp"

	// MaxClockSkew is the validity window for a signed request.
	MaxClockSkew = 60 * time.Second
)

var (
	ErrMissingTimestamp = errors.New("signaling: missing timestamp")
	ErrInvalidTimestamp = errors.New("signaling: invalid timestamp")
	ErrClockSkew        = errors.New("signaling: timestamp outside validity window")
	ErrInvalidSignature = errors.New("signaling: invalid signature")
)

// Sign computes the hex-encoded HMAC-SHA256 of "timestamp.body" using secret.
func Sign(secret []byte, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(strconv.AppendInt(nil, timestamp, 10))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that signatureHex is a valid HMAC-SHA256 over
// "timestampHeader.body" under secret, and that timestampHeader falls within
// MaxClockSkew of now.
func Verify(secret []byte, timestampHeader string, body []byte, signatureHex string, now time.Time) error {
	if timestampHeader == "" {
		return ErrMissingTimestamp
	}

	ts, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimestamp, err)
	}

	skew := now.Sub(time.Unix(ts, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > MaxClockSkew {
		return ErrClockSkew
	}

	want := Sign(secret, ts, body)
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return fmt.Errorf("signaling: encode expected signature: %w", err)
	}
	gotBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("%w: not valid hex", ErrInvalidSignature)
	}

	if !hmac.Equal(wantBytes, gotBytes) {
		return ErrInvalidSignature
	}
	return nil
}
