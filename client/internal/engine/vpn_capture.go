package engine

import (
	"context"
	"log/slog"
	"net"
	"time"

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
	defer router.Teardown(context.Background())

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

func (e *Engine) relayVPNTCP(ctx context.Context, destAddr string, destPort uint16, conn net.Conn) {
	stream, err := e.openRelayStream(destAddrType(destAddr), destAddr, destPort, protocol.ProtoTCP)
	if err != nil {
		conn.Close()
		return
	}
	e.pumpTCP(ctx, conn, stream)
}

func (e *Engine) relayVPNUDP(ctx context.Context, destAddr string, destPort uint16, conn net.Conn) {
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
