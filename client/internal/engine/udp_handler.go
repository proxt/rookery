package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xtaci/smux"

	"github.com/rookery/shared/protocol"
)

// udpStreamIdleTimeout closes a per-destination smux stream if neither
// direction has carried traffic for this long; UDP has no connection-close
// signal of its own.
const udpStreamIdleTimeout = 60 * time.Second

// handleUDPAssociate implements SOCKS5 UDP ASSOCIATE: it opens a local UDP
// relay socket, tells the client where to send datagrams, and forwards them
// through per-destination smux streams until the control connection closes.
func (e *Engine) handleUDPAssociate(ctx context.Context, ctrl net.Conn) {
	relayConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		writeSocks5Reply(ctrl, socksReplyGeneralFailure)
		return
	}
	defer relayConn.Close()

	localAddr := relayConn.LocalAddr().(*net.UDPAddr)
	if err := writeSocks5SuccessAddr(ctrl, localAddr.IP, uint16(localAddr.Port)); err != nil {
		return
	}

	assocCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// SOCKS5 UDP ASSOCIATE carries no further data on the control
	// connection; its closure by the client is what ends the association.
	controlClosed := make(chan struct{})
	go func() {
		defer close(controlClosed)
		buf := make([]byte, 1)
		ctrl.Read(buf)
	}()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-controlClosed:
		case <-ctx.Done():
		case <-done:
			return
		}
		cancel()
		relayConn.Close()
	}()

	a := &udpAssociation{engine: e, relay: relayConn, streams: make(map[string]*udpDestStream)}
	a.run(assocCtx)
}

// udpAssociation demultiplexes datagrams arriving on a single local relay
// socket to one smux stream per distinct destination.
type udpAssociation struct {
	engine *Engine
	relay  *net.UDPConn

	mu         sync.Mutex
	clientAddr *net.UDPAddr
	streams    map[string]*udpDestStream
}

type udpDestStream struct {
	stream   *smux.Stream
	activity chan struct{}
}

func (ds *udpDestStream) poke() {
	select {
	case ds.activity <- struct{}{}:
	default:
	}
}

func (a *udpAssociation) run(ctx context.Context) {
	defer a.closeAllStreams()

	buf := make([]byte, 64*1024)
	for {
		n, addr, err := a.relay.ReadFromUDP(buf)
		if err != nil {
			return
		}

		a.mu.Lock()
		a.clientAddr = addr
		a.mu.Unlock()

		dest, payload, err := decodeSocks5UDPPacket(buf[:n])
		if err != nil {
			continue
		}

		a.forward(ctx, dest, payload)
	}
}

func (a *udpAssociation) closeAllStreams() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ds := range a.streams {
		ds.stream.Close()
	}
}

func (a *udpAssociation) forward(ctx context.Context, dest destination, payload []byte) {
	key := fmt.Sprintf("%d|%s|%d", dest.addrType, dest.addr, dest.port)

	a.mu.Lock()
	ds, ok := a.streams[key]
	a.mu.Unlock()

	if !ok {
		sess := a.engine.sess.Load()
		if sess == nil || sess.IsClosed() {
			return
		}
		stream, err := sess.OpenStream()
		if err != nil {
			return
		}
		header := protocol.Header{AddrType: dest.addrType, Addr: dest.addr, Port: dest.port, Proto: protocol.ProtoUDP}
		if err := protocol.WriteHeader(stream, header); err != nil {
			stream.Close()
			return
		}

		ds = &udpDestStream{stream: stream, activity: make(chan struct{}, 1)}
		a.mu.Lock()
		a.streams[key] = ds
		a.mu.Unlock()
		a.engine.activeStreams.Add(1)
		go a.readReplies(ctx, key, ds, dest)
	}

	if err := protocol.WriteDatagram(ds.stream, payload); err != nil {
		return
	}
	ds.poke()
	a.engine.bytesUp.Add(uint64(len(payload)))
}

// readReplies owns ds's lifetime: it reads datagrams coming back from the
// node, forwards them to the SOCKS5 client, and closes ds on error, ctx
// cancellation, or idle timeout.
func (a *udpAssociation) readReplies(ctx context.Context, key string, ds *udpDestStream, dest destination) {
	defer func() {
		a.mu.Lock()
		delete(a.streams, key)
		a.mu.Unlock()
		ds.stream.Close()
		a.engine.activeStreams.Add(-1)
	}()

	readErr := make(chan struct{}, 1)
	go func() {
		for {
			payload, err := protocol.ReadDatagram(ds.stream)
			if err != nil {
				readErr <- struct{}{}
				return
			}
			ds.poke()

			a.mu.Lock()
			clientAddr := a.clientAddr
			a.mu.Unlock()
			if clientAddr != nil {
				if reply, err := encodeSocks5UDPPacket(dest.addrType, dest.addr, dest.port, payload); err == nil {
					a.relay.WriteToUDP(reply, clientAddr)
					a.engine.bytesDown.Add(uint64(len(payload)))
				}
			}
		}
	}()

	idleTimer := time.NewTimer(udpStreamIdleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-readErr:
			return
		case <-ds.activity:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(udpStreamIdleTimeout)
		case <-idleTimer.C:
			return
		case <-ctx.Done():
			return
		}
	}
}
