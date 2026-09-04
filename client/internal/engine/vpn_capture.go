package engine

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/rookery/client/internal/procmatch"
	"github.com/rookery/client/internal/routing"
	"github.com/rookery/client/internal/vpn"
	"github.com/rookery/shared/protocol"
)

// udpFlowIdleTimeout closes a VPN-mode UDP flow if neither direction has
// carried traffic for this long; UDP has no connection-close signal of its
// own. Matches the SOCKS5 UDP ASSOCIATE path's timeout.
const udpFlowIdleTimeout = 60 * time.Second

// runSystemCapture routes all of the OS's IP traffic through the tunnel via
// a virtual network adapter, instead of requiring each app to be configured
// for the SOCKS5 proxy individually. It blocks until ctx is done.
func (e *Engine) runSystemCapture(ctx context.Context, cfg Config) {
	dev, err := vpn.OpenDevice()
	if err != nil {
		slog.Error("vpn: open device", "error", err)
		return
	}
	defer dev.Close()

	ns, err := vpn.NewNetstack(dev)
	if err != nil {
		slog.Error("vpn: create netstack", "error", err)
		return
	}
	defer ns.Close()

	// Wait for the tunnel to actually be up before taking over the default
	// route — otherwise every app's traffic would black-hole for however
	// long the WebRTC handshake takes.
	if !e.waitConnected(ctx) {
		return
	}

	router, err := vpn.Setup(ctx, cfg.NodeAddr)
	if err != nil {
		slog.Error("vpn: setup routing", "error", err)
		return
	}
	e.router.Store(router)
	defer e.router.Store(nil)
	defer router.Teardown(context.Background())

	// Only once routing has actually taken over the default route can a
	// later drop be a "the tunnel that was carrying traffic just failed"
	// event — gates the kill switch (see killSwitch.routingActive).
	e.setKillSwitchRoutingActive(true)
	defer e.setKillSwitchRoutingActive(false)

	slog.Info("vpn: system-wide capture active")
	defer slog.Info("vpn: system-wide capture stopped")

	err = ns.Run(ctx, e.relayVPNTCP, e.relayVPNUDP)
	if err != nil && ctx.Err() == nil {
		slog.Error("vpn: netstack run", "error", err)
	}
}

// waitConnected blocks until the tunnel reaches StateConnected or ctx is
// done, reporting which happened first.
func (e *Engine) waitConnected(ctx context.Context) bool {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if e.Status().State == StateConnected {
			return true
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return false
		}
	}
}

// dnsPort is the well-known port DNS runs on — flows to it get their
// responses snooped into e.dnsCache regardless of routing decision, so a
// later flow to the resolved IP can still match a domain rule.
const dnsPort = 53

func destAddrType(addr string) protocol.AddrType {
	ip := net.ParseIP(addr)
	switch {
	case ip == nil:
		return protocol.AddrTypeDomain
	case ip.To4() != nil:
		return protocol.AddrTypeIPv4
	default:
		return protocol.AddrTypeIPv6
	}
}

// hostAndIPForVPN resolves what routing.Matcher.Decide needs from a
// VPN-capture destAddr, which is always a bare IP (the TUN device only ever
// sees IP packets, never a domain name) — unlike the SOCKS5 path's
// hostAndIP, which sometimes gets a pre-resolution domain straight from the
// client. host comes from e.dnsCache, populated by snoopDNS on the way past
// for whichever domain (if any) was last resolved to this IP.
func (e *Engine) hostAndIPForVPN(destAddr string) (host string, ip net.IP) {
	ip = net.ParseIP(destAddr)
	if ip == nil {
		return "", nil
	}
	if domain, ok := e.dnsCache.Lookup(ip); ok {
		return domain, ip
	}
	return "", ip
}

// exeNameForTCPPort is relayVPNTCP's counterpart to procmatch.go's
// exeNameForConn — the VPN-capture path gets the app's local port directly
// from gVisor (see stack.go's TCPFlowHandler doc) rather than needing to
// pull it out of a net.Conn.
func exeNameForTCPPort(port uint16) string {
	name, err := procmatch.ExeNameForTCPPort(port)
	if err != nil {
		slog.Debug("engine: exe name lookup (tcp)", "port", port, "error", err)
		return ""
	}
	return name
}

// snoopDNS records a DNS response passing through a UDP:53 flow (either
// direction of the tunneled/direct data path) into e.dnsCache. No-op for
// any other port — cheap to call unconditionally from both relay paths.
func (e *Engine) snoopDNS(destPort uint16, payload []byte) {
	if destPort == dnsPort {
		vpn.SnoopDNSResponse(e.dnsCache, payload)
	}
}

func (e *Engine) relayVPNTCP(ctx context.Context, destAddr string, destPort, srcPort uint16, conn net.Conn) {
	host, ip := e.hostAndIPForVPN(destAddr)
	if e.decide(exeNameForTCPPort(srcPort), host, ip) == routing.ActionDirect {
		e.handleDirectVPNTCP(ctx, conn, destAddr, destPort)
		return
	}

	stream, err := e.openRelayStream(destAddrType(destAddr), destAddr, destPort, protocol.ProtoTCP)
	if err != nil {
		conn.Close()
		return
	}
	e.pumpTCP(ctx, conn, stream)
}

// handleDirectVPNTCP dials destAddr:destPort straight from this machine
// instead of through the tunnel — the VPN-capture counterpart to
// socks_handler.go's handleDirectTCP. Requires a bypass route (via
// e.router) so the dial doesn't loop back into the TUN device, which system
// capture has made the default route.
func (e *Engine) handleDirectVPNTCP(ctx context.Context, conn net.Conn, destAddr string, destPort uint16) {
	router := e.router.Load()
	if router == nil {
		conn.Close()
		return
	}
	if err := router.AddDirectRoute(ctx, destAddr); err != nil {
		slog.Debug("engine: add direct route", "addr", destAddr, "error", err)
		conn.Close()
		return
	}
	defer router.RemoveDirectRoute(context.Background(), destAddr)

	dialCtx, cancel := context.WithTimeout(ctx, directDialTimeout)
	defer cancel()
	target, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort(destAddr, fmt.Sprint(destPort)))
	if err != nil {
		conn.Close()
		return
	}

	e.pumpDirectTCP(ctx, conn, target)
}

func (e *Engine) relayVPNUDP(ctx context.Context, destAddr string, destPort, srcPort uint16, conn net.Conn) {
	host, ip := e.hostAndIPForVPN(destAddr)
	if e.decide(exeNameForUDPPort(int(srcPort)), host, ip) == routing.ActionDirect {
		e.handleDirectVPNUDP(ctx, conn, destAddr, destPort)
		return
	}

	defer conn.Close()

	stream, err := e.openRelayStream(destAddrType(destAddr), destAddr, destPort, protocol.ProtoUDP)
	if err != nil {
		return
	}
	defer stream.Close()

	e.activeStreams.Add(1)
	defer e.activeStreams.Add(-1)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
			stream.Close()
		case <-done:
		}
	}()

	activity := make(chan struct{}, 1)
	poke := func() {
		select {
		case activity <- struct{}{}:
		default:
		}
	}

	errCh := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, protocol.MaxDatagramSize)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- struct{}{}
				return
			}
			if err := protocol.WriteDatagram(stream, buf[:n]); err != nil {
				errCh <- struct{}{}
				return
			}
			e.bytesUp.Add(uint64(n))
			poke()
		}
	}()
	go func() {
		for {
			payload, err := protocol.ReadDatagram(stream)
			if err != nil {
				errCh <- struct{}{}
				return
			}
			e.snoopDNS(destPort, payload)
			if _, err := conn.Write(payload); err != nil {
				errCh <- struct{}{}
				return
			}
			e.bytesDown.Add(uint64(len(payload)))
			poke()
		}
	}()

	idleTimer := time.NewTimer(udpFlowIdleTimeout)
	defer idleTimer.Stop()
	for {
		select {
		case <-errCh:
			return
		case <-activity:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(udpFlowIdleTimeout)
		case <-idleTimer.C:
			return
		}
	}
}

// handleDirectVPNUDP is relayVPNUDP's direct-dial counterpart: a raw UDP
// socket to destAddr:destPort instead of a tunneled smux stream, under the
// same dynamic bypass route as handleDirectVPNTCP.
func (e *Engine) handleDirectVPNUDP(ctx context.Context, conn net.Conn, destAddr string, destPort uint16) {
	defer conn.Close()

	router := e.router.Load()
	if router == nil {
		return
	}
	if err := router.AddDirectRoute(ctx, destAddr); err != nil {
		slog.Debug("engine: add direct route", "addr", destAddr, "error", err)
		return
	}
	defer router.RemoveDirectRoute(context.Background(), destAddr)

	target, err := net.Dial("udp", net.JoinHostPort(destAddr, fmt.Sprint(destPort)))
	if err != nil {
		return
	}
	defer target.Close()

	e.activeStreams.Add(1)
	defer e.activeStreams.Add(-1)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
			target.Close()
		case <-done:
		}
	}()

	activity := make(chan struct{}, 1)
	poke := func() {
		select {
		case activity <- struct{}{}:
		default:
		}
	}

	errCh := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, protocol.MaxDatagramSize)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- struct{}{}
				return
			}
			if _, err := target.Write(buf[:n]); err != nil {
				errCh <- struct{}{}
				return
			}
			e.bytesUp.Add(uint64(n))
			poke()
		}
	}()
	go func() {
		buf := make([]byte, protocol.MaxDatagramSize)
		for {
			n, err := target.Read(buf)
			if err != nil {
				errCh <- struct{}{}
				return
			}
			e.snoopDNS(destPort, buf[:n])
			if _, err := conn.Write(buf[:n]); err != nil {
				errCh <- struct{}{}
				return
			}
			e.bytesDown.Add(uint64(n))
			poke()
		}
	}()

	idleTimer := time.NewTimer(udpFlowIdleTimeout)
	defer idleTimer.Stop()
	for {
		select {
		case <-errCh:
			return
		case <-activity:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(udpFlowIdleTimeout)
		case <-idleTimer.C:
			return
		}
	}
}
