package vpn

import "context"

// Rule names are fixed and predictable on purpose: both engagement and the
// startup cleanup sweep (see CleanupStaleKillSwitchRules) target them by
// exact name, so a rule left over from a crashed previous run is always
// found and removed the same way, regardless of how it got there.
const (
	killSwitchBlockRuleName = "Rookery Kill Switch - Block Outbound"
	killSwitchAllowRuleName = "Rookery Kill Switch - Allow Tunnel"
)

// EngageKillSwitch blocks all outbound traffic except through the tunnel
// interface. The allow rule (scoped to InterfaceName) is created *before*
// the block rule specifically so that if the process dies partway through,
// the worst partial state is "an allow rule that blocks nothing" rather
// than "outbound blocked with no exception carved out yet".
func EngageKillSwitch(ctx context.Context) error {
	if err := runNetsh(ctx, "advfirewall", "firewall", "add", "rule",
		"name="+killSwitchAllowRuleName, "dir=out", "action=allow",
		"interfacealias="+InterfaceName, "enable=yes", "profile=any"); err != nil {
		return err
	}
	return runNetsh(ctx, "advfirewall", "firewall", "add", "rule",
		"name="+killSwitchBlockRuleName, "dir=out", "action=block", "enable=yes", "profile=any")
}

// DisengageKillSwitch removes the kill switch rules, restoring normal
// outbound connectivity. The block rule is removed *first* — symmetric to
// EngageKillSwitch's ordering — so a crash partway through leaves at most
// the harmless allow-only rule, never a still-active block.
func DisengageKillSwitch(ctx context.Context) error {
	err1 := deleteFirewallRule(ctx, killSwitchBlockRuleName)
	err2 := deleteFirewallRule(ctx, killSwitchAllowRuleName)
	if err1 != nil {
		return err1
	}
	return err2
}

// CleanupStaleKillSwitchRules removes any kill switch rules left over from
// an unclean previous shutdown (crash, power loss, taskkill /F) — called
// unconditionally at process startup, before anything else, so a block
// engaged by a run that never got to disengage it doesn't survive past the
// next launch. Identical to DisengageKillSwitch; kept as a separate name so
// call sites read as "safety sweep", not "turning something off".
func CleanupStaleKillSwitchRules(ctx context.Context) error {
	return DisengageKillSwitch(ctx)
}

// deleteFirewallRule reports whatever netsh returns, including "no rules
// match" (the common, harmless case when there's nothing to clean up) —
// callers (engine.killSwitch) log-and-continue rather than treat this as
// fatal, mirroring Router.Teardown's best-effort convention one layer up.
func deleteFirewallRule(ctx context.Context, name string) error {
	return runNetsh(ctx, "advfirewall", "firewall", "delete", "rule", "name="+name)
}
