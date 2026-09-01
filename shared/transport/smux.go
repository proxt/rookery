package transport

import (
	"fmt"
	"io"

	"github.com/xtaci/smux"
)

// NewSmuxClient brings up an smux session as the stream-opening side.
func NewSmuxClient(conn io.ReadWriteCloser) (*smux.Session, error) {
	sess, err := smux.Client(conn, smux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("transport: smux client: %w", err)
	}
	return sess, nil
}

// NewSmuxServer brings up an smux session as the stream-accepting side.
func NewSmuxServer(conn io.ReadWriteCloser) (*smux.Session, error) {
	sess, err := smux.Server(conn, smux.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("transport: smux server: %w", err)
	}
	return sess, nil
}
