package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/rookery/client/internal/subscription"
)

// CachedNode is a node from a subscription's last-known node list, cached
// locally so the UI has something to show before the next refresh.
type CachedNode struct {
	ID      string `json:"id" yaml:"id"`
	Name    string `json:"name" yaml:"name"`
	Tags    string `json:"tags" yaml:"tags"`
	Address string `json:"address" yaml:"address"`
}

// Subscription is one saved rookery://sub/... link: where to fetch its node
// list from, and which of those nodes is currently selected to connect
// through. Session tokens are never persisted — Connect fetches a fresh one
// at the moment it's needed (see App.Connect).
type Subscription struct {
	ID           string       `json:"id" yaml:"id"`
	Name         string       `json:"name" yaml:"name"`
	PanelAddr    string       `json:"panelAddr" yaml:"panel_addr"`
	Token        string       `json:"token" yaml:"token"`
	ActiveNodeID string       `json:"activeNodeId" yaml:"active_node_id"`
	Nodes        []CachedNode `json:"nodes" yaml:"nodes"`
}

// ActiveNode returns the subscription's currently selected node, falling
// back to the first cached node if ActiveNodeID isn't set or no longer
// matches one.
func (s Subscription) ActiveNode() (CachedNode, bool) {
	for _, n := range s.Nodes {
		if n.ID == s.ActiveNodeID {
			return n, true
		}
	}
	if len(s.Nodes) > 0 {
		return s.Nodes[0], true
	}
	return CachedNode{}, false
}

func cacheNodes(nodes []subscription.Node) []CachedNode {
	out := make([]CachedNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, CachedNode{ID: n.ID, Name: n.Name, Tags: n.Tags, Address: n.Address})
	}
	return out
}

// AppSettings is everything the GUI persists: the subscription list plus
// general, subscription-independent settings.
type AppSettings struct {
	Subscriptions        []Subscription `json:"subscriptions" yaml:"subscriptions"`
	ActiveSubscriptionID string         `json:"activeSubscriptionId" yaml:"active_subscription_id"`
	SOCKSPort            int            `json:"socksPort" yaml:"socks_port"`
	AutoStart            bool           `json:"autoStart" yaml:"auto_start"`
	StartMinimized       bool           `json:"startMinimized" yaml:"start_minimized"`
	SystemWide           bool           `json:"systemWide" yaml:"system_wide"`
	// KillSwitch has no effect unless SystemWide is also on. Defaults to
	// off (unlike SystemWide) — it can block all internet access on an
	// unexpected drop, so that's an explicit opt-in, not a surprise.
	KillSwitch bool `json:"killSwitch" yaml:"kill_switch"`
}

// defaultSOCKSPort is used whenever no SOCKS port has been configured yet.
const defaultSOCKSPort = 1080

func defaultAppSettings() AppSettings {
	return AppSettings{Subscriptions: []Subscription{}, SOCKSPort: defaultSOCKSPort, SystemWide: true}
}

// loadSettings reads settings from path. A missing file is not an error —
// it returns defaults instead.
func loadSettings(path string) (AppSettings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultAppSettings(), nil
	}
	if err != nil {
		return AppSettings{}, fmt.Errorf("read settings: %w", err)
	}

	s := defaultAppSettings()
	if err := yaml.Unmarshal(data, &s); err != nil {
		return AppSettings{}, fmt.Errorf("parse settings: %w", err)
	}
	// A YAML file with an empty/absent key unmarshals to a nil slice, which
	// encodes as JSON null — the frontend always expects an array it can
	// call .length/.find on.
	if s.Subscriptions == nil {
		s.Subscriptions = []Subscription{}
	}
	for i := range s.Subscriptions {
		if s.Subscriptions[i].Nodes == nil {
			s.Subscriptions[i].Nodes = []CachedNode{}
		}
	}
	return s, nil
}

func saveSettings(path string, s AppSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// ActiveSubscription returns the currently selected subscription, if any.
func (s AppSettings) ActiveSubscription() (Subscription, bool) {
	for _, sub := range s.Subscriptions {
		if sub.ID == s.ActiveSubscriptionID {
			return sub, true
		}
	}
	return Subscription{}, false
}
