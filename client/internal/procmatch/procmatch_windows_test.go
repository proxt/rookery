package procmatch

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExeNameForTCPPort_SelfProcess is a real end-to-end check, not just a
// compile check: it opens an actual TCP connection from this test binary,
// asks the OS which local port the connection used, and verifies
// ExeNameForTCPPort correctly maps that port back to this same process's
// own executable — the exact thing app-based routing rules depend on.
func TestExeNameForTCPPort_SelfProcess(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c
		}
	}()

	client, err := net.Dial("tcp4", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	server := <-accepted
	defer server.Close()

	localPort := uint16(client.LocalAddr().(*net.TCPAddr).Port)

	got, err := ExeNameForTCPPort(localPort)
	if err != nil {
		t.Fatalf("ExeNameForTCPPort: %v", err)
	}
	if got == "" {
		t.Fatal("ExeNameForTCPPort returned empty — expected this test process's own exe name")
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	want := filepath.Base(self)

	if !strings.EqualFold(got, want) {
		t.Errorf("got exe %q, want %q (this test binary)", got, want)
	}
}

func TestExeNameForUDPPort_SelfProcess(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer conn.Close()

	localPort := uint16(conn.LocalAddr().(*net.UDPAddr).Port)

	got, err := ExeNameForUDPPort(localPort)
	if err != nil {
		t.Fatalf("ExeNameForUDPPort: %v", err)
	}
	if got == "" {
		t.Fatal("ExeNameForUDPPort returned empty — expected this test process's own exe name")
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	want := filepath.Base(self)

	if !strings.EqualFold(got, want) {
		t.Errorf("got exe %q, want %q (this test binary)", got, want)
	}
}

func TestExeNameForTCPPort_UnusedPortReturnsEmpty(t *testing.T) {
	// Port 1 is reserved and essentially never actually bound — the table
	// scan should just not find it, not error.
	got, err := ExeNameForTCPPort(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty for an unbound port", got)
	}
}
