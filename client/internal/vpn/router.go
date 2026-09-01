package vpn

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
)

// Router owns the system-wide routing changes made while the tunnel is
// active. Teardown must be called to restore normal networking.
type Router struct {
	nodeIP           string
	dnsPolicyChanged bool
}

// Setup points the system's default route at the tunnel interface, after
// first carving out an explicit bypass route for nodeAddr's resolved IP so
// the tunnel's own signaling traffic isn't captured by the tunnel it's still
// in the middle of building. It also disables Windows' "smart multi-homed
// name resolution", which otherwise queries every active adapter's DNS
// server (including the pre-tunnel one) and races their answers — a classic
// DNS leak even once the tunnel is the preferred route.
func Setup(ctx context.Context, nodeAddr string) (*Router, error) {
	nodeIP, err := resolveHost(nodeAddr)
	if err != nil {
		return nil, fmt.Errorf("vpn: resolve node address: %w", err)
	}

	gw, err := currentDefaultGateway(ctx)
	if err != nil {
		return nil, err
	}

	if err := addBypassRoute(ctx, nodeIP, gw); err != nil {
		return nil, fmt.Errorf("vpn: add bypass route: %w", err)
	}

	if err := configureInterface(ctx); err != nil {
		removeBypassRoute(ctx, nodeIP)
		return nil, err
	}

	if err := addDefaultRoute(ctx); err != nil {
		removeBypassRoute(ctx, nodeIP)
		return nil, fmt.Errorf("vpn: add default route: %w", err)
	}

	dnsChanged, err := disableSmartMultiHomedResolution()
	if err != nil {
		slog.Warn("vpn: disable smart multi-homed name resolution", "error", err)
	} else if dnsChanged {
		flushDNSCache(ctx)
	}

	return &Router{nodeIP: nodeIP, dnsPolicyChanged: dnsChanged}, nil
}

// Teardown restores the system's normal routing. Safe to call more than
// once; each step is best-effort so a failure in one doesn't block the rest.
func (r *Router) Teardown(ctx context.Context) {
	if err := removeDefaultRoute(ctx); err != nil {
		slog.Warn("vpn: remove default route", "error", err)
	}
	if err := removeBypassRoute(ctx, r.nodeIP); err != nil {
		slog.Warn("vpn: remove bypass route", "error", err)
	}
	if r.dnsPolicyChanged {
		if err := restoreSmartMultiHomedResolution(); err != nil {
			slog.Warn("vpn: restore smart multi-homed name resolution", "error", err)
		}
		flushDNSCache(ctx)
	}
}

func resolveHost(nodeAddr string) (string, error) {
	u, err := url.Parse(nodeAddr)
	if err != nil {
		return "", fmt.Errorf("parse node address: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		host = nodeAddr
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("lookup %s: %w", host, err)
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found for %s", host)
}
