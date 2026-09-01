package vpn

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// noWindow suppresses the console window Windows would otherwise flash for
// each spawned netsh/route/powershell child, since rookery-gui.exe itself
// has no console of its own to attach them to.
var noWindow = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW

// defaultGateway is the system's current default route: the interface and
// next-hop everything used before the tunnel took over. Captured before any
// change is made, so it can be restored exactly.
type defaultGateway struct {
	nextHop string
}

// currentDefaultGateway asks Windows for the highest-priority IPv4 default
// route's next hop.
func currentDefaultGateway(ctx context.Context) (defaultGateway, error) {
	out, err := runPowerShell(ctx,
		`(Get-NetRoute -DestinationPrefix 0.0.0.0/0 -AddressFamily IPv4 | `+
			`Sort-Object -Property RouteMetric | Select-Object -First 1 -ExpandProperty NextHop)`)
	if err != nil {
		return defaultGateway{}, fmt.Errorf("vpn: query default gateway: %w", err)
	}

	nextHop := strings.TrimSpace(out)
	if net.ParseIP(nextHop) == nil {
		return defaultGateway{}, fmt.Errorf("vpn: no usable default gateway found")
	}
	return defaultGateway{nextHop: nextHop}, nil
}

// configureInterface assigns InterfaceIP/DNSServer to the Wintun adapter and
// forces its own interface metric to the lowest value.
//
// Windows picks a route by interface-metric + route-metric combined, not by
// route metric alone. Wintun adapters get a mediocre automatic interface
// metric, which can lose to a fast Ethernet/Wi-Fi adapter's low automatic
// metric even after we add a route with metric=1 — the route "wins" on
// paper but the physical adapter's route still has the lower total cost, so
// traffic keeps flowing there. Setting the interface's own metric to 1
// removes that ambiguity.
func configureInterface(ctx context.Context) error {
	if err := runNetsh(ctx, "interface", "ip", "set", "address",
		"name="+InterfaceName, "static", InterfaceIP, InterfaceMask); err != nil {
		return fmt.Errorf("vpn: set interface address: %w", err)
	}
	if err := runNetsh(ctx, "interface", "ip", "set", "dns",
		"name="+InterfaceName, "static", DNSServer); err != nil {
		return fmt.Errorf("vpn: set interface dns: %w", err)
	}
	if err := runNetsh(ctx, "interface", "ipv4", "set", "interface",
		"interface="+InterfaceName, "metric=1"); err != nil {
		return fmt.Errorf("vpn: set interface metric: %w", err)
	}
	return nil
}

// bypassRoute adds a host route sending nodeIP through the original default
// gateway, so the tunnel's own signaling/DataChannel traffic to the node
// doesn't try to route through the tunnel it's building (which would be a
// loop: the tunnel can't come up until this traffic gets through).
func addBypassRoute(ctx context.Context, nodeIP string, gw defaultGateway) error {
	return runRoute(ctx, "add", nodeIP, "mask", "255.255.255.255", gw.nextHop, "metric", "1")
}

func removeBypassRoute(ctx context.Context, nodeIP string) error {
	return runRoute(ctx, "delete", nodeIP)
}

// addDefaultRoute makes the tunnel the system's default route.
func addDefaultRoute(ctx context.Context) error {
	return runNetsh(ctx, "interface", "ipv4", "add", "route", "prefix=0.0.0.0/0",
		"interface="+InterfaceName, "metric=1", "store=active")
}

func removeDefaultRoute(ctx context.Context) error {
	return runNetsh(ctx, "interface", "ipv4", "delete", "route", "prefix=0.0.0.0/0",
		"interface="+InterfaceName, "store=active")
}

func runNetsh(ctx context.Context, args ...string) error {
	return runCommand(ctx, "netsh", args...)
}

// flushDNSCache clears cached DNS answers so a just-changed resolution
// policy or DNS server takes effect immediately instead of after the old
// entries' TTL expires. Best-effort by design — callers just log failures.
func flushDNSCache(ctx context.Context) {
	runCommand(ctx, "ipconfig", "/flushdns")
}

func runRoute(ctx context.Context, args ...string) error {
	return runCommand(ctx, "route", args...)
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, name, args...)
	cmd.SysProcAttr = noWindow
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func runPowerShell(ctx context.Context, script string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = noWindow
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("powershell: %w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
