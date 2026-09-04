package routing

import (
	"encoding/json"
	"fmt"
	"strings"
)

// v2ray-core's routing config shape (the subset this cares about) — see
// https://www.v2fly.org/config/routing.html. Rookery only ever emits/reads
// the "field" rule type, one Rookery Rule per v2ray rule, which keeps the
// mapping one-to-one and lossless for domain/geoip rules.
type v2rayConfig struct {
	Routing v2rayRouting `json:"routing"`
}

type v2rayRouting struct {
	DomainStrategy string      `json:"domainStrategy,omitempty"`
	Rules          []v2rayRule `json:"rules"`
}

type v2rayRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

// ToV2rayRoutingJSON renders set as a v2ray-core routing config. Rules of
// Type app have no equivalent in v2ray (v2ray-core isn't OS-level — it has
// no concept of "which process opened this connection"), so they're
// skipped; skipped is the count of rules left out, for the caller to
// surface to the user ("N rules were not exported").
func ToV2rayRoutingJSON(set RuleSet) (data []byte, skipped int, err error) {
	cfg := v2rayConfig{Routing: v2rayRouting{DomainStrategy: "AsIs"}}
	for _, r := range set.Rules {
		tag := "proxy"
		if r.Action == ActionDirect {
			tag = "direct"
		}
		switch r.Type {
		case RuleTypeDomain:
			cfg.Routing.Rules = append(cfg.Routing.Rules, v2rayRule{
				Type: "field", Domain: []string{"domain:" + r.Value}, OutboundTag: tag,
			})
		case RuleTypeGeoIP:
			cfg.Routing.Rules = append(cfg.Routing.Rules, v2rayRule{
				Type: "field", IP: []string{"geoip:" + strings.ToLower(r.Value)}, OutboundTag: tag,
			})
		default:
			skipped++
		}
	}
	data, err = json.MarshalIndent(cfg, "", "  ")
	return data, skipped, err
}

// FromV2rayRoutingJSON parses a v2ray-core config (or just its top-level
// "routing" object) and returns the domain/geoip rules it can represent.
// geosite: entries and raw IP/CIDR entries (as opposed to geoip:<country>)
// have no equivalent in Rookery's rule model and are skipped, counted the
// same way ToV2rayRoutingJSON counts app rules it can't export.
func FromV2rayRoutingJSON(data []byte, name string) (set RuleSet, skipped int, err error) {
	var cfg v2rayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Also accept a bare {"domainStrategy":...,"rules":[...]} object,
		// i.e. just the "routing" value without the wrapping config.
		var routing v2rayRouting
		if err2 := json.Unmarshal(data, &routing); err2 != nil {
			return RuleSet{}, 0, fmt.Errorf("routing: parse v2ray config: %w", err)
		}
		cfg.Routing = routing
	}

	set = RuleSet{Name: name}
	for _, vr := range cfg.Routing.Rules {
		action := ActionProxy
		if vr.OutboundTag == "direct" {
			action = ActionDirect
		}
		for _, d := range vr.Domain {
			v, ok := strings.CutPrefix(d, "domain:")
			if !ok {
				skipped++
				continue
			}
			set.Rules = append(set.Rules, Rule{Type: RuleTypeDomain, Value: v, Action: action})
		}
		for _, ip := range vr.IP {
			v, ok := strings.CutPrefix(ip, "geoip:")
			if !ok {
				skipped++
				continue
			}
			set.Rules = append(set.Rules, Rule{Type: RuleTypeGeoIP, Value: strings.ToUpper(v), Action: action})
		}
	}
	return set, skipped, nil
}
