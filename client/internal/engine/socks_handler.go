package engine

import (
	"context"
	"io"
	"net"
	"sync/atomic"

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
	defer conn.Close()

	dest, err := socks5Handshake(conn)
	if err != nil {
		return
	}

	if dest.cmd == socksCmdUDPAssociate {
		e.handleUDPAssociate(ctx, conn)
		return
	}

	sess := e.sess.Load()
	if sess == nil || sess.IsClosed() {
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		return
	}

	stream, err := sess.OpenStream()
	if err != nil {
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		return
	}
	defer stream.Close()

	header := protocol.Header{AddrType: dest.addrType, Addr: dest.addr, Port: dest.port, Proto: protocol.ProtoTCP}
	if err := protocol.WriteHeader(stream, header); err != nil {
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		return
	}

	if err := writeSocks5Success(conn); err != nil {
		return
	}

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
