package engine

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/xtaci/smux"

	"github.com/rookery/client/internal/routing"
	"github.com/rookery/shared/protocol"
)

// directDialTimeout bounds a direct (non-tunneled) TCP dial, mirroring the
// kind of timeout a normal browser/OS connect attempt would use.
const directDialTimeout = 10 * time.Second

// hostAndIP splits a SOCKS5/protocol destination into the (host, ip) pair
// routing.Matcher.Decide wants: a domain destination gives a hostname and no
// IP (GeoIP can't apply pre-resolution — a known, accepted limitation, see
// routing package docs); an IP-literal destination gives no hostname and a
// parsed net.IP.
func hostAndIP(addrType protocol.AddrType, addr string) (host string, ip net.IP) {
	if addrType == protocol.AddrTypeDomain {
		return addr, nil
	}
	return "", net.ParseIP(addr)
}

func (e *Engine) acceptSOCKS(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			e.handleSOCKSConn(ctx, conn)
		}()
	}
}

func (e *Engine) handleSOCKSConn(ctx context.Context, conn net.Conn) {
	dest, err := socks5Handshake(conn)
	if err != nil {
		conn.Close()
		return
	}

	if dest.cmd == socksCmdUDPAssociate {
		e.handleUDPAssociate(ctx, conn)
		return
	}

	host, ip := hostAndIP(dest.addrType, dest.addr)
	if e.decide("", host, ip) == routing.ActionDirect {
		e.handleDirectTCP(ctx, conn, dest)
		return
	}

	stream, err := e.openRelayStream(dest.addrType, dest.addr, dest.port, protocol.ProtoTCP)
	if err != nil {
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		conn.Close()
		return
	}

	if err := writeSocks5Success(conn); err != nil {
		stream.Close()
		conn.Close()
		return
	}

	e.pumpTCP(ctx, conn, stream)
}

// handleDirectTCP dials dest straight from this machine, bypassing the
// tunnel — the routing.ActionDirect path. Mirrors the tunneled path above
// (same SOCKS5 reply handshake, same pumpDirectTCP byte-counting) so a
// direct connection looks identical to the SOCKS5 client either way.
func (e *Engine) handleDirectTCP(ctx context.Context, conn net.Conn, dest destination) {
	dialCtx, cancel := context.WithTimeout(ctx, directDialTimeout)
	defer cancel()

	dialer := net.Dialer{}
	target, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(dest.addr, fmt.Sprint(dest.port)))
	if err != nil {
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		conn.Close()
		return
	}

	if err := writeSocks5Success(conn); err != nil {
		target.Close()
		conn.Close()
		return
	}

	e.pumpDirectTCP(ctx, conn, target)
}

// pumpDirectTCP is pumpTCP's counterpart for a direct (non-tunneled)
// connection — same activeStreams/byte-counter bookkeeping so the
// dashboard's traffic totals include direct traffic too, just without a
// smux stream in the middle.
func (e *Engine) pumpDirectTCP(ctx context.Context, conn, target net.Conn) {
	defer conn.Close()
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

	doneCh := make(chan struct{}, 2)
	go func() {
		copyAndCount(target, conn, &e.bytesUp)
		doneCh <- struct{}{}
	}()
	go func() {
		copyAndCount(conn, target, &e.bytesDown)
		doneCh <- struct{}{}
	}()
	<-doneCh
}

// openRelayStream opens a new smux stream to the node and writes its
// destination header. The caller owns the returned stream.
func (e *Engine) openRelayStream(addrType protocol.AddrType, addr string, port uint16, proto protocol.Proto) (*smux.Stream, error) {
	sess := e.sess.Load()
	if sess == nil || sess.IsClosed() {
		return nil, fmt.Errorf("engine: no active session")
	}

	stream, err := sess.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("engine: open stream: %w", err)
	}

	header := protocol.Header{AddrType: addrType, Addr: addr, Port: port, Proto: proto}
	if err := protocol.WriteHeader(stream, header); err != nil {
		stream.Close()
		return nil, fmt.Errorf("engine: write header: %w", err)
	}
	return stream, nil
}

// pumpTCP bidirectionally copies bytes between conn and stream, tracking
// activeStreams and byte counters, until either side closes or ctx is done.
// It closes both conn and stream before returning.
func (e *Engine) pumpTCP(ctx context.Context, conn net.Conn, stream *smux.Stream) {
	defer conn.Close()
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

	doneCh := make(chan struct{}, 2)
	go func() {
		copyAndCount(stream, conn, &e.bytesUp)
		doneCh <- struct{}{}
	}()
	go func() {
		copyAndCount(conn, stream, &e.bytesDown)
		doneCh <- struct{}{}
	}()
	<-doneCh
}

// copyAndCount copies from src to dst, adding every byte moved to counter.
func copyAndCount(dst io.Writer, src io.Reader, counter *atomic.Uint64) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
			counter.Add(uint64(n))
		}
		if err != nil {
			return
		}
	}
}
