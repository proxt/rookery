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

	"github.com/rookery/client/internal/routing"
)

// httpTimeout bounds a single fetch.
const httpTimeout = 10 * time.Second

// pingTimeout bounds a single node latency probe — short, since it's UI
// feedback for a node-picker list and one dead node shouldn't stall it.
const pingTimeout = 2 * time.Second

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
	// RoutingRuleSet is the panel-assigned routing rule set for this
	// subscription's user, if one is assigned — nil otherwise. The JSON
	// shape matches routing.RuleSet exactly (see panel's subapi handlers),
	// so it decodes directly with no intermediate DTO.
	RoutingRuleSet *routing.RuleSet `json:"routing_rule_set,omitempty"`
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

// MeasurePing probes address's /ping endpoint and returns the round-trip
// time. Result is intentionally not part of Node/Subscription — latency is
// transient and measured client-side after a fetch, never persisted.
func MeasurePing(ctx context.Context, address string) (time.Duration, error) {
	url := strings.TrimSuffix(address, "/") + "/ping"

	reqCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("subscription: build ping request: %w", err)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("subscription: ping: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("subscription: node returned status %d", resp.StatusCode)
	}
	return elapsed, nil
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
