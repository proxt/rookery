package routing

import (
	"net"
	"testing"
)

func TestMatchDomainSuffix(t *testing.T) {
	m := NewMatcher(RuleSet{Rules: []Rule{
		{Type: RuleTypeDomain, Value: "google.com", Action: ActionDirect},
	}})

	cases := map[string]Action{
		"google.com":     ActionDirect,
		"www.google.com": ActionDirect,
		"mail.google.com": ActionDirect,
		"notgoogle.com":  ActionProxy,
		"googlecom":      ActionProxy,
		"example.com":    ActionProxy,
	}
	for host, want := range cases {
		if got := m.Decide("", host, nil); got != want {
			t.Errorf("Decide(%q): got %s, want %s", host, got, want)
		}
	}
}

func TestMatchApp(t *testing.T) {
	m := NewMatcher(RuleSet{Rules: []Rule{
		{Type: RuleTypeApp, Value: "chrome.exe", Action: ActionDirect},
	}})
	if got := m.Decide("chrome.exe", "", nil); got != ActionDirect {
		t.Errorf("got %s, want direct", got)
	}
	if got := m.Decide("Chrome.EXE", "", nil); got != ActionDirect {
		t.Errorf("case-insensitive match failed: got %s", got)
	}
	if got := m.Decide("firefox.exe", "", nil); got != ActionProxy {
		t.Errorf("got %s, want proxy (no match)", got)
	}
}

func TestMatchGeoIP(t *testing.T) {
	// 1.0.1.0 is CN in the embedded dbip-country table (checked against the
	// source CSV at generation time — see geoipdata/countries_ipv4.bin).
	m := NewMatcher(RuleSet{Rules: []Rule{
		{Type: RuleTypeGeoIP, Value: "CN", Action: ActionDirect},
	}})
	if got := m.Decide("", "", net.ParseIP("1.0.1.5")); got != ActionDirect {
		t.Errorf("got %s, want direct for a CN-geoip IP", got)
	}
	if got := m.Decide("", "", net.ParseIP("8.8.8.8")); got != ActionProxy {
		t.Errorf("got %s, want proxy for a non-CN IP", got)
	}
}

func TestLocalRuleSetWinsOverLater(t *testing.T) {
	local := RuleSet{Name: "local", Rules: []Rule{
		{Type: RuleTypeDomain, Value: "example.com", Action: ActionDirect},
	}}
	fromPanel := RuleSet{Name: "panel", Rules: []Rule{
		{Type: RuleTypeDomain, Value: "example.com", Action: ActionProxy},
	}}
	// Caller puts local first — Matcher itself has no special-casing, it
	// just takes the first match in set order, so this also documents the
	// contract callers (app.go's buildMatcher) must follow.
	m := NewMatcher(local, fromPanel)
	if got := m.Decide("", "example.com", nil); got != ActionDirect {
		t.Errorf("got %s, want the local set's direct to win", got)
	}
}

func TestNoRulesDefaultsToProxy(t *testing.T) {
	m := NewMatcher()
	if got := m.Decide("anything.exe", "example.com", net.ParseIP("1.2.3.4")); got != ActionProxy {
		t.Errorf("got %s, want proxy as the safe default", got)
	}
}
