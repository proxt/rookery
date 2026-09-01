package profile

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		l    Link
	}{
		{"basic", Link{NodeAddr: "https://vpn.example.com", UserID: "abc123", Secret: "s3cr3t"}},
		{"with-name", Link{NodeAddr: "https://vpn.example.com", UserID: "abc123", Secret: "s3cr3t", Name: "My Laptop"}},
		{"name-with-special-chars", Link{NodeAddr: "https://vpn.example.com", UserID: "u1", Secret: "s1", Name: "Home & Office #1"}},
		{"unicode-name", Link{NodeAddr: "https://vpn.example.com", UserID: "u1", Secret: "s1", Name: "Ноутбук"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			link := Encode(tc.l)
			got, err := Decode(link)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got != tc.l {
				t.Fatalf("round trip mismatch: got %+v, want %+v", got, tc.l)
			}
		})
	}
}

func TestDecodeGarbageInput(t *testing.T) {
	cases := []struct {
		name    string
		link    string
		wantErr error
	}{
		{"empty", "", ErrEmptyLink},
		{"whitespace-only", "   ", ErrEmptyLink},
		{"wrong-scheme", "vmess://eyJhIjoxfQ", ErrInvalidScheme},
		{"no-scheme", "eyJhIjoxfQ", ErrInvalidScheme},
		{"not-base64", "rookery://not-valid-base64!!!", nil},
		{"base64-but-not-json", "rookery://" + "bm90LWpzb24", nil},
		{"missing-fields", "rookery://" + encodeRaw(`{"n":"https://x.com"}`), nil},
		{"empty-json", "rookery://" + encodeRaw(`{}`), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.link)
			if err == nil {
				t.Fatalf("Decode(%q) expected error, got nil", tc.link)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Decode(%q) error = %v, want %v", tc.link, err, tc.wantErr)
			}
		})
	}
}

func TestDecodeTrimsWhitespace(t *testing.T) {
	link := Encode(Link{NodeAddr: "https://vpn.example.com", UserID: "u1", Secret: "s1"})
	got, err := Decode("  " + link + "\n")
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.NodeAddr != "https://vpn.example.com" {
		t.Fatalf("got %+v", got)
	}
}

func encodeRaw(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
