// Package routing decides, per destination, whether traffic should go
// through the Rookery tunnel or straight out the machine's own network —
// driven by named rule sets that can come from a subscription's panel or be
// authored locally in the client (see Matcher).
package routing

// Action is what a matching rule decides for a destination.
type Action string

const (
	// ActionProxy sends the destination through the tunnel — the default
	// for anything no rule matches, since that's what a VPN client is for.
	ActionProxy Action = "proxy"
	// ActionDirect dials the destination straight from the local machine,
	// bypassing the tunnel entirely.
	ActionDirect Action = "direct"
)

// RuleType is what a Rule matches against.
type RuleType string

const (
	// RuleTypeDomain matches a hostname by exact match or suffix (so
	// "google.com" also matches "www.google.com").
	RuleTypeDomain RuleType = "domain"
	// RuleTypeApp matches the executable name (e.g. "chrome.exe") of the
	// process that originated the connection.
	RuleTypeApp RuleType = "app"
	// RuleTypeGeoIP matches the destination IP's country, as a two-letter
	// code (e.g. "RU"), resolved via the embedded GeoIP table (see geoip.go).
	RuleTypeGeoIP RuleType = "geoip"
)

// Rule is one routing decision: destinations matching Type/Value get Action.
type Rule struct {
	ID     string   `json:"id"`
	Type   RuleType `json:"type"`
	Value  string   `json:"value"`
	Action Action   `json:"action"`
}

// RuleSet is a named, ordered collection of rules — what both the panel and
// the client manage as one unit (create/edit/delete/import/export), matching
// how subscriptions are already a named list rather than one flat set of
// servers.
type RuleSet struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}
