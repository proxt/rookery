package vpn

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sync"
)

// Router owns the system-wide routing changes made while the tunnel is
// active. Teardown must be called to restore normal networking.
type Router struct {
	nodeIP           string
	dnsPolicyChanged bool
	gw               defaultGateway

	// directMu/directRefs back AddDirectRoute/RemoveDirectRoute: per-
	// destination bypass routes added on demand for routing.ActionDirect
	// traffic under system-wide capture, ref-counted so two concurrent
	// flows to the same IP don't have the second flow's close remove the
	// route out from under the first.
	directMu   sync.Mutex
	directRefs map[string]int
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

	return &Router{nodeIP: nodeIP, dnsPolicyChanged: dnsChanged, gw: gw, directRefs: make(map[string]int)}, nil
}

// AddDirectRoute ensures ip bypasses the tunnel via the original default
// gateway — the dynamic counterpart to the fixed bypass route Setup adds for
// the node itself, used for routing.ActionDirect destinations under
// system-wide capture. Ref-counted: concurrent flows to the same IP share
// one route, removed only once the last of them calls RemoveDirectRoute.
func (r *Router) AddDirectRoute(ctx context.Context, ip string) error {
	r.directMu.Lock()
	defer r.directMu.Unlock()

	if r.directRefs[ip] > 0 {
		r.directRefs[ip]++
		return nil
	}
	if err := addBypassRoute(ctx, ip, r.gw); err != nil {
		return fmt.Errorf("vpn: add direct route for %s: %w", ip, err)
	}
	r.directRefs[ip] = 1
	return nil
}

// RemoveDirectRoute releases one reference added by AddDirectRoute, removing
// the bypass route once nothing else is using it. Best-effort, like the rest
// of Router's teardown steps.
func (r *Router) RemoveDirectRoute(ctx context.Context, ip string) {
	r.directMu.Lock()
	defer r.directMu.Unlock()

	if r.directRefs[ip] == 0 {
		return
	}
	r.directRefs[ip]--
	if r.directRefs[ip] > 0 {
		return
	}
	delete(r.directRefs, ip)
	if err := removeBypassRoute(ctx, ip); err != nil {
		slog.Warn("vpn: remove direct route", "ip", ip, "error", err)
	}
}

// Teardown restores the system's normal routing. Safe to call more than
// once; each step is best-effort so a failure in one doesn't block the rest.
func (r *Router) Teardown(ctx context.Context) {
	r.directMu.Lock()
	for ip := range r.directRefs {
		if err := removeBypassRoute(ctx, ip); err != nil {
			slog.Warn("vpn: remove direct route", "ip", ip, "error", err)
		}
	}
	r.directRefs = make(map[string]int)
	r.directMu.Unlock()

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
