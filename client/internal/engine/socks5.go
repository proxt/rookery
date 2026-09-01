package engine

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/rookery/shared/protocol"
)

const (
	socksVersion5 = 0x05

	socksCmdConnect      = 0x01
	socksCmdUDPAssociate = 0x03

	socksAtypIPv4   = 0x01
	socksAtypDomain = 0x03
	socksAtypIPv6   = 0x04

	socksAuthNone     = 0x00
	socksAuthNoAccept = 0xFF

	socksReplySucceeded         = 0x00
	socksReplyGeneralFailure    = 0x01
	socksReplyCommandNotSupport = 0x07
)

// destination is what a SOCKS5 request asked for, already in protocol.Header
// terms, plus which command was requested.
type destination struct {
	cmd      byte
	addrType protocol.AddrType
	addr     string
	port     uint16
}

// socks5Handshake performs the greeting and request phases of a SOCKS5
// connection (no authentication; CONNECT and UDP ASSOCIATE only) and returns
// the requested destination.
func socks5Handshake(conn net.Conn) (destination, error) {
	if err := socks5Greeting(conn); err != nil {
		return destination{}, err
	}
	return socks5Request(conn)
}

func socks5Greeting(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return fmt.Errorf("socks5: read greeting: %w", err)
	}
	if header[0] != socksVersion5 {
		return fmt.Errorf("socks5: unsupported version 0x%02x", header[0])
	}

	methods := make([]byte, header[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("socks5: read methods: %w", err)
	}

	hasNoAuth := false
	for _, m := range methods {
		if m == socksAuthNone {
			hasNoAuth = true
			break
		}
	}
	if !hasNoAuth {
		conn.Write([]byte{socksVersion5, socksAuthNoAccept})
		return fmt.Errorf("socks5: client offered no acceptable auth method")
	}

	if _, err := conn.Write([]byte{socksVersion5, socksAuthNone}); err != nil {
		return fmt.Errorf("socks5: write greeting reply: %w", err)
	}
	return nil
}

func socks5Request(conn net.Conn) (destination, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return destination{}, fmt.Errorf("socks5: read request: %w", err)
	}
	if header[0] != socksVersion5 {
		return destination{}, fmt.Errorf("socks5: unsupported version 0x%02x", header[0])
	}
	cmd := header[1]
	atyp := header[3]

	var addr string
	var addrType protocol.AddrType
	switch atyp {
	case socksAtypIPv4:
		b := make([]byte, 4)
		if _, err := io.ReadFull(conn, b); err != nil {
			return destination{}, fmt.Errorf("socks5: read ipv4 address: %w", err)
		}
		addr = net.IP(b).String()
		addrType = protocol.AddrTypeIPv4
	case socksAtypIPv6:
		b := make([]byte, 16)
		if _, err := io.ReadFull(conn, b); err != nil {
			return destination{}, fmt.Errorf("socks5: read ipv6 address: %w", err)
		}
		addr = net.IP(b).String()
		addrType = protocol.AddrTypeIPv6
	case socksAtypDomain:
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenByte); err != nil {
			return destination{}, fmt.Errorf("socks5: read domain length: %w", err)
		}
		b := make([]byte, lenByte[0])
		if _, err := io.ReadFull(conn, b); err != nil {
			return destination{}, fmt.Errorf("socks5: read domain: %w", err)
		}
		addr = string(b)
		addrType = protocol.AddrTypeDomain
	default:
		writeSocks5Reply(conn, socksReplyGeneralFailure)
		return destination{}, fmt.Errorf("socks5: unsupported address type 0x%02x", atyp)
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return destination{}, fmt.Errorf("socks5: read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBytes)

	if cmd != socksCmdConnect && cmd != socksCmdUDPAssociate {
		writeSocks5Reply(conn, socksReplyCommandNotSupport)
		return destination{}, fmt.Errorf("socks5: unsupported command 0x%02x", cmd)
	}

	return destination{cmd: cmd, addrType: addrType, addr: addr, port: port}, nil
}

func writeSocks5Reply(conn net.Conn, rep byte) {
	conn.Write([]byte{socksVersion5, rep, 0x00, socksAtypIPv4, 0, 0, 0, 0, 0, 0})
}

func writeSocks5Success(conn net.Conn) error {
	_, err := conn.Write([]byte{socksVersion5, socksReplySucceeded, 0x00, socksAtypIPv4, 0, 0, 0, 0, 0, 0})
	return err
}

// writeSocks5SuccessAddr replies success with a specific BND.ADDR/BND.PORT,
// as SOCKS5 UDP ASSOCIATE requires (it tells the client where to send its
// UDP datagrams).
func writeSocks5SuccessAddr(conn net.Conn, ip net.IP, port uint16) error {
	atyp := byte(socksAtypIPv4)
	addrBytes := ip.To4()
	if addrBytes == nil {
		atyp = socksAtypIPv6
		addrBytes = ip.To16()
	}

	buf := make([]byte, 0, 6+len(addrBytes))
	buf = append(buf, socksVersion5, socksReplySucceeded, 0x00, atyp)
	buf = append(buf, addrBytes...)
	buf = binary.BigEndian.AppendUint16(buf, port)

	_, err := conn.Write(buf)
	return err
}

// decodeSocks5UDPPacket parses a SOCKS5 UDP request datagram: [RSV(2)]
// [FRAG(1)][ATYP][DST.ADDR][DST.PORT][DATA]. Fragmentation is not supported.
func decodeSocks5UDPPacket(data []byte) (dest destination, payload []byte, err error) {
	if len(data) < 4 {
		return destination{}, nil, fmt.Errorf("socks5udp: packet too short")
	}
	if data[2] != 0 {
		return destination{}, nil, fmt.Errorf("socks5udp: fragmentation not supported")
	}

	atyp := data[3]
	i := 4

	var addr string
	var addrType protocol.AddrType
	switch atyp {
	case socksAtypIPv4:
		if len(data) < i+4 {
			return destination{}, nil, fmt.Errorf("socks5udp: truncated ipv4 address")
		}
		addr = net.IP(data[i : i+4]).String()
		addrType = protocol.AddrTypeIPv4
		i += 4
	case socksAtypIPv6:
		if len(data) < i+16 {
			return destination{}, nil, fmt.Errorf("socks5udp: truncated ipv6 address")
		}
		addr = net.IP(data[i : i+16]).String()
		addrType = protocol.AddrTypeIPv6
		i += 16
	case socksAtypDomain:
		if len(data) < i+1 {
			return destination{}, nil, fmt.Errorf("socks5udp: truncated domain length")
		}
		n := int(data[i])
		i++
		if len(data) < i+n {
			return destination{}, nil, fmt.Errorf("socks5udp: truncated domain")
		}
		addr = string(data[i : i+n])
		addrType = protocol.AddrTypeDomain
		i += n
	default:
		return destination{}, nil, fmt.Errorf("socks5udp: unsupported address type 0x%02x", atyp)
	}

	if len(data) < i+2 {
		return destination{}, nil, fmt.Errorf("socks5udp: truncated port")
	}
	port := binary.BigEndian.Uint16(data[i : i+2])
	i += 2

	return destination{cmd: socksCmdUDPAssociate, addrType: addrType, addr: addr, port: port}, data[i:], nil
}

// encodeSocks5UDPPacket builds a SOCKS5 UDP reply datagram carrying payload
// as having come from addrType/addr/port.
func encodeSocks5UDPPacket(addrType protocol.AddrType, addr string, port uint16, payload []byte) ([]byte, error) {
	var atyp byte
	var addrBytes []byte

	switch addrType {
	case protocol.AddrTypeIPv4:
		ip := net.ParseIP(addr).To4()
		if ip == nil {
			return nil, fmt.Errorf("socks5udp: invalid ipv4 address %q", addr)
		}
		atyp = socksAtypIPv4
		addrBytes = ip
	case protocol.AddrTypeIPv6:
		ip := net.ParseIP(addr).To16()
		if ip == nil {
			return nil, fmt.Errorf("socks5udp: invalid ipv6 address %q", addr)
		}
		atyp = socksAtypIPv6
		addrBytes = ip
	case protocol.AddrTypeDomain:
		if len(addr) > 255 {
			return nil, fmt.Errorf("socks5udp: domain too long")
		}
		atyp = socksAtypDomain
		addrBytes = append([]byte{byte(len(addr))}, []byte(addr)...)
	default:
		return nil, fmt.Errorf("socks5udp: unknown address type %v", addrType)
	}

	buf := make([]byte, 0, 4+len(addrBytes)+2+len(payload))
	buf = append(buf, 0, 0, 0, atyp)
	buf = append(buf, addrBytes...)
	buf = binary.BigEndian.AppendUint16(buf, port)
	buf = append(buf, payload...)
	return buf, nil
}
