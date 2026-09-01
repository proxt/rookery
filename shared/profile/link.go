// Package profile defines the rookery:// link that hands a client everything
// it needs to connect: node address, user ID, and per-user secret.
package profile

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Scheme is the URI scheme rookery:// links use.
const Scheme = "rookery"

var (
	ErrInvalidScheme = errors.New("profile: link must start with rookery://")
	ErrEmptyLink     = errors.New("profile: link is empty")
)

// Link is a connection profile: where the node is, and which user's
// credentials to authenticate with.
type Link struct {
	NodeAddr string
	UserID   string
	Secret   string
	Name     string
}

type payload struct {
	NodeAddr string `json:"n"`
	UserID   string `json:"i"`
	Secret   string `json:"s"`
}

// Encode builds a rookery:// link from l.
func Encode(l Link) string {
	p := payload{NodeAddr: l.NodeAddr, UserID: l.UserID, Secret: l.Secret}
	data, _ := json.Marshal(p) // payload is plain strings; cannot fail

	u := Scheme + "://" + base64.RawURLEncoding.EncodeToString(data)
	if l.Name != "" {
		u += "#" + url.QueryEscape(l.Name)
	}
	return u
}

// Decode parses a rookery:// link.
func Decode(link string) (Link, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return Link{}, ErrEmptyLink
	}

	prefix := Scheme + "://"
	if !strings.HasPrefix(link, prefix) {
		return Link{}, ErrInvalidScheme
	}
	rest := strings.TrimPrefix(link, prefix)

	var name string
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		frag := rest[i+1:]
		rest = rest[:i]
		if decoded, err := url.QueryUnescape(frag); err == nil {
			name = decoded
		} else {
			name = frag
		}
	}

	data, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return Link{}, fmt.Errorf("profile: decode link: %w", err)
	}

	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return Link{}, fmt.Errorf("profile: parse link: %w", err)
	}

	if p.NodeAddr == "" || p.UserID == "" || p.Secret == "" {
		return Link{}, fmt.Errorf("profile: link is missing required fields")
	}

	return Link{NodeAddr: p.NodeAddr, UserID: p.UserID, Secret: p.Secret, Name: name}, nil
}
