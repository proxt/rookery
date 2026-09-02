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
		{"basic", Link{PanelAddr: "https://panel.example.com", Token: "abc123"}},
		{"different-token", Link{PanelAddr: "https://panel.example.com", Token: "s3cr3ttoken"}},
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

func TestEncodeUsesSubPrefix(t *testing.T) {
	link := Encode(Link{PanelAddr: "https://panel.example.com", Token: "abc123"})
	if want := "rookery://sub/"; link[:len(want)] != want {
		t.Fatalf("Encode() = %q, want prefix %q", link, want)
	}
}

func TestDecodeAcceptsPlainSubURL(t *testing.T) {
	cases := []struct {
		name string
		link string
		want Link
	}{
		{"basic", "https://panel.example.com/sub/abc123", Link{PanelAddr: "https://panel.example.com", Token: "abc123"}},
		{"trailing-slash", "https://panel.example.com/sub/abc123/", Link{PanelAddr: "https://panel.example.com", Token: "abc123"}},
		{"http-scheme", "http://panel.example.com/sub/abc123", Link{PanelAddr: "http://panel.example.com", Token: "abc123"}},
		{"with-port", "https://panel.example.com:8090/sub/abc123", Link{PanelAddr: "https://panel.example.com:8090", Token: "abc123"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode(tc.link)
			if err != nil {
				t.Fatalf("Decode(%q) error = %v", tc.link, err)
			}
			if got != tc.want {
				t.Fatalf("Decode(%q) = %+v, want %+v", tc.link, got, tc.want)
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
		{"old-scheme-no-sub", "rookery://eyJhIjoxfQ", ErrInvalidScheme},
		{"url-without-sub-path", "https://panel.example.com/admin", ErrInvalidScheme},
		{"url-with-empty-token", "https://panel.example.com/sub/", ErrInvalidScheme},
		{"not-base64", "rookery://sub/not-valid-base64!!!", nil},
		{"base64-but-not-json", "rookery://sub/" + "bm90LWpzb24", nil},
		{"missing-fields", "rookery://sub/" + encodeRaw(`{"p":"https://x.com"}`), nil},
		{"empty-json", "rookery://sub/" + encodeRaw(`{}`), nil},
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
	link := Encode(Link{PanelAddr: "https://panel.example.com", Token: "u1"})
	got, err := Decode("  " + link + "\n")
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.PanelAddr != "https://panel.example.com" {
		t.Fatalf("got %+v", got)
	}
}

func encodeRaw(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
