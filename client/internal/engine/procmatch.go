package engine

import (
	"log/slog"
	"net"

	"github.com/rookery/client/internal/procmatch"
)

// exeNameForConn looks up which process owns the SOCKS5 client side of
// conn, for app-based routing rules. Best-effort: any lookup failure (the
// process exited between accept and lookup, or the table scan itself
// errored) just means "no app rule applies", logged at debug level rather
// than surfaced — this must never block a connection from proceeding.
func (e *Engine) exeNameForConn(conn net.Conn) string {
	addr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return ""
	}
	name, err := procmatch.ExeNameForTCPPort(uint16(addr.Port))
	if err != nil {
		slog.Debug("engine: exe name lookup (tcp)", "port", addr.Port, "error", err)
		return ""
	}
	return name
}

// exeNameForUDPPort is exeNameForConn's counterpart for the SOCKS5 UDP
// ASSOCIATE path, where the relevant local port is the client's UDP relay
// source port rather than a net.Conn.
func exeNameForUDPPort(port int) string {
	if port <= 0 || port > 65535 {
		return ""
	}
	name, err := procmatch.ExeNameForUDPPort(uint16(port))
	if err != nil {
		slog.Debug("engine: exe name lookup (udp)", "port", port, "error", err)
		return ""
	}
	return name
}
