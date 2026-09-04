package routing

import (
	"net"
	"strings"
)

// Matcher decides direct-vs-proxy for a destination against an ordered list
// of rule sets. Order encodes priority: earlier sets win on conflict — the
// caller is responsible for putting the client's own local rule sets before
// whatever a subscription delivered (see app.go's buildMatcher), matching
// the product decision that local rules always take precedence.
type Matcher struct {
	sets []RuleSet
}

// NewMatcher builds a Matcher from sets, in priority order (earlier wins).
// A nil/disabled set should simply be omitted by the caller rather than
// passed in empty — Matcher itself has no notion of "enabled".
func NewMatcher(sets ...RuleSet) *Matcher {
	return &Matcher{sets: sets}
}

// Decide is the single entry point every relay path calls: given what's
// known about a flow (the process that opened it, if identified; the
// hostname, if known — pre-resolution on the SOCKS5 path, or via DNS
// sniffing under system-wide capture; and the destination IP, if resolved),
// it returns the action to take. Precedence, most to least specific: an app
// rule ("route this whole program") beats a domain rule ("route this one
// site") beats a GeoIP rule ("route this whole country") beats the default
// of ActionProxy — an unmatched destination always goes through the tunnel,
// never leaks direct by accident.
func (m *Matcher) Decide(exeName, host string, ip net.IP) Action {
	if exeName != "" {
		if a, ok := m.matchApp(exeName); ok {
			return a
		}
	}
	if host != "" {
		if a, ok := m.matchDomain(host); ok {
			return a
		}
	}
	if ip != nil {
		if a, ok := m.matchGeoIP(ip); ok {
			return a
		}
	}
	return ActionProxy
}

func (m *Matcher) matchApp(exeName string) (Action, bool) {
	exeName = strings.ToLower(exeName)
	for _, set := range m.sets {
		for _, r := range set.Rules {
			if r.Type == RuleTypeApp && strings.ToLower(r.Value) == exeName {
				return r.Action, true
			}
		}
	}
	return "", false
}

// matchDomain does exact or suffix matching: a rule for "google.com" also
// matches "www.google.com" but not "notgoogle.com".
func (m *Matcher) matchDomain(host string) (Action, bool) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, set := range m.sets {
		for _, r := range set.Rules {
			if r.Type != RuleTypeDomain {
				continue
			}
			rv := strings.ToLower(strings.TrimPrefix(r.Value, "."))
			if host == rv || strings.HasSuffix(host, "."+rv) {
				return r.Action, true
			}
		}
	}
	return "", false
}

func (m *Matcher) matchGeoIP(ip net.IP) (Action, bool) {
	code := countryFor(ip)
	if code == "" {
		return "", false
	}
	for _, set := range m.sets {
		for _, r := range set.Rules {
			if r.Type == RuleTypeGeoIP && strings.EqualFold(r.Value, code) {
				return r.Action, true
			}
		}
	}
	return "", false
}
