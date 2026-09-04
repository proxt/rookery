package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xtaci/smux"

	"github.com/rookery/client/internal/routing"
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

	a := &udpAssociation{
		engine:      e,
		relay:       relayConn,
		streams:     make(map[string]*udpDestStream),
		directDests: make(map[string]*udpDirectDest),
	}
	a.run(assocCtx)
}

// udpAssociation demultiplexes datagrams arriving on a single local relay
// socket to one smux stream per distinct destination.
type udpAssociation struct {
	engine *Engine
	relay  *net.UDPConn

	mu          sync.Mutex
	clientAddr  *net.UDPAddr
	streams     map[string]*udpDestStream
	directDests map[string]*udpDirectDest
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

// udpDirectDest is a direct (non-tunneled) UDP "connection" — routing.
// ActionDirect's counterpart to udpDestStream. A UDP socket is already
// message-oriented, so unlike the tunneled path there's no
// protocol.WriteDatagram/ReadDatagram framing to apply: payloads go
// straight in and out of conn.
type udpDirectDest struct {
	conn     net.Conn
	activity chan struct{}
}

func (dd *udpDirectDest) poke() {
	select {
	case dd.activity <- struct{}{}:
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
	for _, dd := range a.directDests {
		dd.conn.Close()
	}
}

func (a *udpAssociation) forward(ctx context.Context, dest destination, payload []byte) {
	key := fmt.Sprintf("%d|%s|%d", dest.addrType, dest.addr, dest.port)

	a.mu.Lock()
	clientPort := 0
	if a.clientAddr != nil {
		clientPort = a.clientAddr.Port
	}
	a.mu.Unlock()

	host, ip := hostAndIP(dest.addrType, dest.addr)
	if a.engine.decide(exeNameForUDPPort(clientPort), host, ip) == routing.ActionDirect {
		a.forwardDirect(ctx, key, dest, payload)
		return
	}

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

// forwardDirect is forward's counterpart for routing.ActionDirect
// destinations: dials the real destination straight from this machine
// instead of opening a tunnel stream.
func (a *udpAssociation) forwardDirect(ctx context.Context, key string, dest destination, payload []byte) {
	a.mu.Lock()
	dd, ok := a.directDests[key]
	a.mu.Unlock()

	if !ok {
		conn, err := net.Dial("udp", net.JoinHostPort(dest.addr, fmt.Sprint(dest.port)))
		if err != nil {
			return
		}

		dd = &udpDirectDest{conn: conn, activity: make(chan struct{}, 1)}
		a.mu.Lock()
		a.directDests[key] = dd
		a.mu.Unlock()
		a.engine.activeStreams.Add(1)
		go a.readDirectReplies(ctx, key, dd, dest)
	}

	if _, err := dd.conn.Write(payload); err != nil {
		return
	}
	dd.poke()
	a.engine.bytesUp.Add(uint64(len(payload)))
}

// readDirectReplies is readReplies' counterpart for a direct UDP
// destination — no protocol.ReadDatagram framing, just raw reads off the
// socket.
func (a *udpAssociation) readDirectReplies(ctx context.Context, key string, dd *udpDirectDest, dest destination) {
	defer func() {
		a.mu.Lock()
		delete(a.directDests, key)
		a.mu.Unlock()
		dd.conn.Close()
		a.engine.activeStreams.Add(-1)
	}()

	readErr := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, protocol.MaxDatagramSize)
		for {
			n, err := dd.conn.Read(buf)
			if err != nil {
				readErr <- struct{}{}
				return
			}
			dd.poke()

			a.mu.Lock()
			clientAddr := a.clientAddr
			a.mu.Unlock()
			if clientAddr != nil {
				if reply, err := encodeSocks5UDPPacket(dest.addrType, dest.addr, dest.port, buf[:n]); err == nil {
					a.relay.WriteToUDP(reply, clientAddr)
					a.engine.bytesDown.Add(uint64(n))
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
		case <-dd.activity:
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
