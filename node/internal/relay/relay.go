// Package relay dials a stream's requested destination and pipes bytes
// between it and the smux stream that carried the request.
package relay

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/rookery/shared/protocol"
)

// udpIdleTimeout closes a UDP relay if neither direction has carried traffic
// for this long; UDP has no connection-close signal of its own.
const udpIdleTimeout = 60 * time.Second

// Config controls dial behavior for relayed streams.
type Config struct {
	DialTimeout     time.Duration
	AllowPrivateNet bool
}

// Handle reads the destination header from stream, dials it, and relays
// traffic between it and stream until either side closes or ctx is
// canceled. It always closes stream before returning.
func Handle(ctx context.Context, stream io.ReadWriteCloser, cfg Config) {
	defer stream.Close()

	header, err := protocol.ReadHeader(stream)
	if err != nil {
		slog.Warn("relay: read header", "error", err)
		return
	}

	dialCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	ip, err := resolveAndCheck(dialCtx, header.Addr, cfg.AllowPrivateNet)
	cancel()
	if err != nil {
		slog.Warn("relay: resolve destination", "addr", header.Addr, "error", err)
		return
	}

	switch header.Proto {
	case protocol.ProtoTCP:
		handleTCP(ctx, stream, ip, header)
	case protocol.ProtoUDP:
		handleUDP(ctx, stream, ip, header)
	default:
		slog.Warn("relay: unsupported protocol for this stream type", "proto", header.Proto)
	}
}

func handleTCP(ctx context.Context, stream io.ReadWriteCloser, ip string, header protocol.Header) {
	dest := net.JoinHostPort(ip, strconv.Itoa(int(header.Port)))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", dest)
	if err != nil {
		slog.Warn("relay: dial failed", "addr", dest, "error", err)
		return
	}
	defer conn.Close()

	slog.Info("relay: tcp stream open", "requested_addr", header.Addr, "dial_addr", dest)
	defer slog.Info("relay: tcp stream closed", "dial_addr", dest)

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			stream.Close()
			conn.Close()
		case <-done:
		}
	}()

	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, stream)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(stream, conn)
		errCh <- err
	}()
	<-errCh
}

func handleUDP(ctx context.Context, stream io.ReadWriteCloser, ip string, header protocol.Header) {
	udpAddr := &net.UDPAddr{IP: net.ParseIP(ip), Port: int(header.Port)}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		slog.Warn("relay: udp dial failed", "addr", udpAddr, "error", err)
		return
	}
	defer conn.Close()
	defer stream.Close()

	slog.Info("relay: udp stream open", "requested_addr", header.Addr, "dial_addr", udpAddr)
	defer slog.Info("relay: udp stream closed", "dial_addr", udpAddr)

	activity := make(chan struct{}, 1)
	poke := func() {
		select {
		case activity <- struct{}{}:
		default:
		}
	}

	errCh := make(chan struct{}, 2)

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
			poke()
		}
	}()

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
			poke()
		}
	}()

	idleTimer := time.NewTimer(udpIdleTimeout)
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
			idleTimer.Reset(udpIdleTimeout)
		case <-idleTimer.C:
			return
		case <-ctx.Done():
			return
		}
	}
}

// resolveAndCheck resolves addr once and, unless allowPrivate is set,
// rejects it if any resolved address is private. The resolved IP (rather
// than addr itself) is what gets dialed, so a later DNS response can't
// rebind the destination after the check has passed.
func resolveAndCheck(ctx context.Context, addr string, allowPrivate bool) (string, error) {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
	if err != nil {
		return "", fmt.Errorf("lookup: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no addresses found for %q", addr)
	}

	if !allowPrivate {
		for _, ip := range ips {
			if IsPrivate(ip.IP) {
				return "", fmt.Errorf("%q resolves to disallowed private address %s", addr, ip.IP)
			}
		}
	}

	return ips[0].IP.String(), nil
}
