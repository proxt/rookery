// Package profile defines the rookery:// link that hands a client
// everything it needs to add a subscription: the panel to fetch it from,
// and the subscription's own token.
package profile

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Scheme is the URI scheme rookery:// links use.
const Scheme = "rookery"

// subPrefix is the full prefix a subscription link starts with:
// "rookery://sub/".
const subPrefix = Scheme + "://sub/"

var (
	ErrInvalidScheme = errors.New("profile: link must start with rookery://sub/")
	ErrEmptyLink     = errors.New("profile: link is empty")
)

// Link points a client at a subscription: fetch PanelAddr + "/sub/" + Token
// (or just resolve it directly) to get the current node list.
type Link struct {
	PanelAddr string
	Token     string
}

type payload struct {
	PanelAddr string `json:"p"`
	Token     string `json:"t"`
}

// Encode builds a rookery://sub/... link from l.
func Encode(l Link) string {
	p := payload{PanelAddr: l.PanelAddr, Token: l.Token}
	data, _ := json.Marshal(p) // payload is plain strings; cannot fail
	return subPrefix + base64.RawURLEncoding.EncodeToString(data)
}

// Decode parses a rookery://sub/... link.
func Decode(link string) (Link, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return Link{}, ErrEmptyLink
	}

	if !strings.HasPrefix(link, subPrefix) {
		return Link{}, ErrInvalidScheme
	}
	rest := strings.TrimPrefix(link, subPrefix)

	data, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return Link{}, fmt.Errorf("profile: decode link: %w", err)
	}

	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return Link{}, fmt.Errorf("profile: parse link: %w", err)
	}

	if p.PanelAddr == "" || p.Token == "" {
		return Link{}, fmt.Errorf("profile: link is missing required fields")
	}

	return Link{PanelAddr: p.PanelAddr, Token: p.Token}, nil
}
