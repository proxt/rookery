// Package profile defines the rookery:// link that hands a client
// everything it needs to add a subscription: the panel to fetch it from,
// and the subscription's own token.
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

// Decode parses either a rookery://sub/... deep link, or the plain
// https://panel/sub/{token} URL it's derived from — the sub page's browser
// address bar and its "copy link" button both hand out the latter, so a
// client has to accept whatever a user actually copies.
func Decode(link string) (Link, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return Link{}, ErrEmptyLink
	}

	if strings.HasPrefix(link, subPrefix) {
		return decodeSubLink(link)
	}
	if l, err := decodeSubURL(link); err == nil {
		return l, nil
	}
	return Link{}, ErrInvalidScheme
}

func decodeSubLink(link string) (Link, error) {
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

// decodeSubURL parses "https://panel.example.com/sub/{token}" (with an
// optional trailing slash) into a Link.
func decodeSubURL(link string) (Link, error) {
	u, err := url.Parse(link)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Link{}, ErrInvalidScheme
	}

	const marker = "/sub/"
	i := strings.LastIndex(u.Path, marker)
	if i < 0 {
		return Link{}, ErrInvalidScheme
	}
	token := strings.Trim(u.Path[i+len(marker):], "/")
	if token == "" {
		return Link{}, ErrInvalidScheme
	}

	panelAddr := u.Scheme + "://" + u.Host + u.Path[:i]
	return Link{PanelAddr: panelAddr, Token: token}, nil
}
