// Package subscription resolves a panel subscription (panel address + token,
// as decoded from a rookery://sub/... link) into the list of nodes it
// currently grants access to, each with a freshly signed session token.
package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// httpTimeout bounds a single fetch.
const httpTimeout = 10 * time.Second

// Node is one relay server a subscription grants access to.
type Node struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Tags         string `json:"tags"`
	Address      string `json:"address"`
	SessionToken string `json:"session_token"`
}

// Subscription is what the panel's /sub/{token} endpoint returns.
type Subscription struct {
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
}

// Fetch resolves a subscription by calling panelAddr's /sub/{token}
// endpoint. It always asks for JSON explicitly (the endpoint otherwise
// serves an HTML landing page for browsers).
func Fetch(ctx context.Context, panelAddr, token string) (Subscription, error) {
	url := strings.TrimSuffix(panelAddr, "/") + "/sub/" + token

	reqCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return Subscription{}, fmt.Errorf("subscription: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Subscription{}, fmt.Errorf("subscription: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Subscription{}, fmt.Errorf("subscription: panel returned status %d", resp.StatusCode)
	}

	var sub Subscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return Subscription{}, fmt.Errorf("subscription: decode response: %w", err)
	}
	return sub, nil
}

// FindNode returns the node with the given ID, if present.
func (s Subscription) FindNode(id string) (Node, bool) {
	for _, n := range s.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}
