package routing

import "testing"

func TestV2rayRoundTrip(t *testing.T) {
	orig := RuleSet{Name: "mine", Rules: []Rule{
		{Type: RuleTypeDomain, Value: "google.com", Action: ActionDirect},
		{Type: RuleTypeGeoIP, Value: "RU", Action: ActionProxy},
		{Type: RuleTypeApp, Value: "chrome.exe", Action: ActionDirect}, // not exportable
	}}

	data, skipped, err := ToV2rayRoutingJSON(orig)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("expected 1 skipped (app rule), got %d", skipped)
	}

	got, skipped2, err := FromV2rayRoutingJSON(data, "mine")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if skipped2 != 0 {
		t.Fatalf("expected 0 skipped on import, got %d", skipped2)
	}
	if len(got.Rules) != 2 {
		t.Fatalf("expected 2 rules back, got %d: %+v", len(got.Rules), got.Rules)
	}
	if got.Rules[0].Type != RuleTypeDomain || got.Rules[0].Value != "google.com" || got.Rules[0].Action != ActionDirect {
		t.Errorf("domain rule round-trip mismatch: %+v", got.Rules[0])
	}
	if got.Rules[1].Type != RuleTypeGeoIP || got.Rules[1].Value != "RU" || got.Rules[1].Action != ActionProxy {
		t.Errorf("geoip rule round-trip mismatch: %+v", got.Rules[1])
	}
}

func TestFromV2rayRoutingJSONSkipsGeosite(t *testing.T) {
	data := []byte(`{"routing":{"rules":[{"type":"field","domain":["geosite:google"],"outboundTag":"proxy"}]}}`)
	set, skipped, err := FromV2rayRoutingJSON(data, "x")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if skipped != 1 || len(set.Rules) != 0 {
		t.Fatalf("expected geosite: entry to be skipped, got skipped=%d rules=%+v", skipped, set.Rules)
	}
}
