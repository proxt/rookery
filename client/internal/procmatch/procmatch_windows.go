// Package procmatch answers "which process owns this local TCP/UDP port?" —
// the missing piece for app-based routing rules, since neither the SOCKS5
// protocol nor a captured IP packet ever says which program originated a
// connection. Windows exposes this via iphlpapi's GetExtendedTcpTable/
// GetExtendedUdpTable (owner-PID variant); there's no wrapper for either in
// golang.org/x/sys/windows, so this binds them directly.
package procmatch

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modiphlpapi             = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtendedTcpTable = modiphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable = modiphlpapi.NewProc("GetExtendedUdpTable")
)

const (
	afINET = 2 // AF_INET — IPv4 only; app rules don't cover IPv6 sockets.

	tcpTableOwnerPIDAll = 5 // TCP_TABLE_OWNER_PID_ALL
	udpTableOwnerPID    = 1 // UDP_TABLE_OWNER_PID

	errInsufficientBuffer = 122 // ERROR_INSUFFICIENT_BUFFER
)

// mibTCPRowOwnerPID mirrors MIB_TCPROW_OWNER_PID. dwLocalPort/dwRemotePort
// hold the port in network byte order packed into the low 16 bits of a
// 32-bit field — see localPort()/portFromWire below.
type mibTCPRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

type mibUDPRowOwnerPID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

// portFromWire unpacks a GetExtendedTcpTable/UdpTable port field: the real
// port is stored big-endian in the field's low 2 bytes (high 2 bytes
// unused), which a little-endian machine loads byte-swapped relative to
// what we want — so swap them back.
func portFromWire(wire uint32) uint16 {
	return uint16(wire&0xFF)<<8 | uint16((wire>>8)&0xFF)
}

// tcpOwnerByPort returns the PID owning the given local TCP port (IPv4,
// any local address), or 0 if not found.
func tcpOwnerByPort(port uint16) (uint32, error) {
	var size uint32
	// First call sizes the buffer (expected to fail with
	// ERROR_INSUFFICIENT_BUFFER — that's not a real error here).
	ret, _, _ := procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, afINET, tcpTableOwnerPIDAll, 0)
	if ret != 0 && ret != errInsufficientBuffer {
		return 0, fmt.Errorf("procmatch: size GetExtendedTcpTable: %d", ret)
	}
	if size == 0 {
		return 0, nil
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, afINET, tcpTableOwnerPIDAll, 0,
	)
	if ret != 0 {
		return 0, fmt.Errorf("procmatch: GetExtendedTcpTable: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(mibTCPRowOwnerPID{})
	base := unsafe.Pointer(&buf[4]) // skip dwNumEntries
	for i := uint32(0); i < numEntries; i++ {
		row := (*mibTCPRowOwnerPID)(unsafe.Add(base, uintptr(i)*rowSize))
		if portFromWire(row.LocalPort) == port {
			return row.OwningPID, nil
		}
	}
	return 0, nil
}

// udpOwnerByPort is tcpOwnerByPort's UDP counterpart.
func udpOwnerByPort(port uint16) (uint32, error) {
	var size uint32
	ret, _, _ := procGetExtendedUdpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, afINET, udpTableOwnerPID, 0)
	if ret != 0 && ret != errInsufficientBuffer {
		return 0, fmt.Errorf("procmatch: size GetExtendedUdpTable: %d", ret)
	}
	if size == 0 {
		return 0, nil
	}

	buf := make([]byte, size)
	ret, _, _ = procGetExtendedUdpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, afINET, udpTableOwnerPID, 0,
	)
	if ret != 0 {
		return 0, fmt.Errorf("procmatch: GetExtendedUdpTable: %d", ret)
	}

	numEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowSize := unsafe.Sizeof(mibUDPRowOwnerPID{})
	base := unsafe.Pointer(&buf[4])
	for i := uint32(0); i < numEntries; i++ {
		row := (*mibUDPRowOwnerPID)(unsafe.Add(base, uintptr(i)*rowSize))
		if portFromWire(row.LocalPort) == port {
			return row.OwningPID, nil
		}
	}
	return 0, nil
}

// exeNameForPID resolves a PID to its executable's base filename (e.g.
// "chrome.exe"), matching the convention rule values are written in.
func exeNameForPID(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("procmatch: open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", fmt.Errorf("procmatch: query image name for pid %d: %w", pid, err)
	}
	full := windows.UTF16ToString(buf[:size])
	return filepath.Base(full), nil
}

// ExeNameForTCPPort returns the executable name (e.g. "chrome.exe") of the
// process holding localPort open for an outbound TCP connection — the port
// a SOCKS5 client connected *from* (conn.RemoteAddr() as seen by our
// listener), not the destination port. Returns "" (not an error) if the
// port isn't found — a benign race (the connection can close between
// accept and this lookup) that callers should treat as "unknown, no app
// rule applies" rather than a failure.
func ExeNameForTCPPort(localPort uint16) (string, error) {
	pid, err := tcpOwnerByPort(localPort)
	if err != nil {
		return "", err
	}
	if pid == 0 {
		return "", nil
	}
	name, err := exeNameForPID(pid)
	if err != nil {
		// The process may have exited between the table snapshot and the
		// OpenProcess call — not fatal, just unresolvable this time.
		return "", nil
	}
	return name, nil
}

// ExeNameForUDPPort is ExeNameForTCPPort's UDP counterpart, for the SOCKS5
// UDP ASSOCIATE path.
func ExeNameForUDPPort(localPort uint16) (string, error) {
	pid, err := udpOwnerByPort(localPort)
	if err != nil {
		return "", err
	}
	if pid == 0 {
		return "", nil
	}
	name, err := exeNameForPID(pid)
	if err != nil {
		return "", nil
	}
	return name, nil
}
