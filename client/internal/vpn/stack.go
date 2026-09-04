package vpn

import (
	"context"
	"fmt"
	"net"

	wgtun "golang.zx2c4.com/wireguard/tun"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const nicID = tcpip.NICID(1)

// TCPFlowHandler is called for each new intercepted TCP connection. conn
// behaves like a net.Conn; the destination the local app was trying to
// reach is destAddr:destPort, and srcPort is the local port the app itself
// used to open the connection (for app-based routing — it's the same local
// port Windows' TCP table reports for that process). The handler owns conn
// and must close it.
type TCPFlowHandler func(ctx context.Context, destAddr string, destPort, srcPort uint16, conn net.Conn)

// UDPFlowHandler is called for each new intercepted UDP flow. Same contract
// as TCPFlowHandler.
type UDPFlowHandler func(ctx context.Context, destAddr string, destPort, srcPort uint16, conn net.Conn)

// Netstack terminates every TCP/UDP flow the OS routes to the TUN device
// (via promiscuous + spoofing mode, so it isn't limited to intercepting
// traffic addressed to its own interface address) and hands each one to a
// handler instead of routing it further.
type Netstack struct {
	dev   wgtun.Device
	ep    *channel.Endpoint
	stack *stack.Stack
}

// NewNetstack wires dev into a gVisor network stack.
func NewNetstack(dev wgtun.Device) (*Netstack, error) {
	ep := channel.New(1024, MTU, "")

	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})

	if err := s.CreateNIC(nicID, ep); err != nil {
		return nil, fmt.Errorf("vpn: create nic: %s", err)
	}
	// Promiscuous + spoofing: intercept and originate traffic for addresses
	// other than the NIC's own — this is what lets one adapter capture
	// connections to arbitrary internet destinations instead of just itself.
	if err := s.SetPromiscuousMode(nicID, true); err != nil {
		return nil, fmt.Errorf("vpn: set promiscuous mode: %s", err)
	}
	if err := s.SetSpoofing(nicID, true); err != nil {
		return nil, fmt.Errorf("vpn: set spoofing: %s", err)
	}

	addr := tcpip.AddrFromSlice(net.ParseIP(InterfaceIP).To4())
	protoAddr := tcpip.ProtocolAddress{Protocol: ipv4.ProtocolNumber, AddressWithPrefix: addr.WithPrefix()}
	if err := s.AddProtocolAddress(nicID, protoAddr, stack.AddressProperties{}); err != nil {
		return nil, fmt.Errorf("vpn: add protocol address: %s", err)
	}

	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	return &Netstack{dev: dev, ep: ep, stack: s}, nil
}

// Run pumps packets between the real TUN device and the netstack, and
// dispatches every intercepted TCP/UDP flow to onTCP/onUDP. It blocks until
// ctx is canceled or the TUN device errors.
func (n *Netstack) Run(ctx context.Context, onTCP TCPFlowHandler, onUDP UDPFlowHandler) error {
	fwdTCP := tcp.NewForwarder(n.stack, 0, 512, func(r *tcp.ForwarderRequest) {
		id := r.ID()
		var wq waiter.Queue
		ep, err := r.CreateEndpoint(&wq)
		if err != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)
		onTCP(ctx, id.LocalAddress.String(), id.LocalPort, id.RemotePort, gonet.NewTCPConn(&wq, ep))
	})
	n.stack.SetTransportProtocolHandler(tcp.ProtocolNumber, fwdTCP.HandlePacket)

	fwdUDP := udp.NewForwarder(n.stack, func(r *udp.ForwarderRequest) {
		id := r.ID()
		var wq waiter.Queue
		ep, err := r.CreateEndpoint(&wq)
		if err != nil {
			return
		}
		go onUDP(ctx, id.LocalAddress.String(), id.LocalPort, id.RemotePort, gonet.NewUDPConn(&wq, ep))
	})
	n.stack.SetTransportProtocolHandler(udp.ProtocolNumber, fwdUDP.HandlePacket)

	errCh := make(chan error, 2)
	go n.pumpInbound(ctx, errCh)
	go n.pumpOutbound(ctx, errCh)

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// pumpInbound reads raw packets from the real TUN device and injects them
// into the netstack.
func (n *Netstack) pumpInbound(ctx context.Context, errCh chan<- error) {
	batchSize := n.dev.BatchSize()
	bufs := make([][]byte, batchSize)
	for i := range bufs {
		bufs[i] = make([]byte, MTU+16)
	}
	sizes := make([]int, batchSize)

	for {
		if ctx.Err() != nil {
			return
		}

		count, err := n.dev.Read(bufs, sizes, 0)
		if err != nil {
			select {
			case errCh <- fmt.Errorf("vpn: read tun: %w", err):
			default:
			}
			return
		}

		for i := 0; i < count; i++ {
			packet := bufs[i][:sizes[i]]
			if len(packet) == 0 {
				continue
			}

			var proto tcpip.NetworkProtocolNumber
			switch packet[0] >> 4 {
			case 4:
				proto = ipv4.ProtocolNumber
			case 6:
				proto = ipv6.ProtocolNumber
			default:
				continue
			}

			pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
				Payload: buffer.MakeWithData(append([]byte(nil), packet...)),
			})
			n.ep.InjectInbound(proto, pkt)
			pkt.DecRef()
		}
	}
}

// pumpOutbound reads packets the netstack wants to send and writes them to
// the real TUN device.
func (n *Netstack) pumpOutbound(ctx context.Context, errCh chan<- error) {
	for {
		pkt := n.ep.ReadContext(ctx)
		if pkt == nil {
			return // ctx canceled
		}

		buf := pkt.ToBuffer()
		data := buf.Flatten()
		pkt.DecRef()

		if _, err := n.dev.Write([][]byte{data}, 0); err != nil {
			select {
			case errCh <- fmt.Errorf("vpn: write tun: %w", err):
			default:
			}
			return
		}
	}
}

// Close tears down the netstack (but not the underlying TUN device, which
// the caller owns and must close separately).
func (n *Netstack) Close() {
	n.stack.Close()
}
