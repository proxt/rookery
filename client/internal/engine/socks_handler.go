package engine

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"github.com/xtaci/smux"

	"github.com/rookery/shared/protocol"
)

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
