package dnscache

import (
	"net"
	"testing"
	"time"
)

func TestSetAndLookup(t *testing.T) {
	c := New()
	ip := net.ParseIP("93.184.216.34")
	c.Set(ip, "example.com", time.Minute)

	domain, ok := c.Lookup(ip)
	if !ok || domain != "example.com" {
		t.Fatalf("Lookup() = (%q, %v), want (\"example.com\", true)", domain, ok)
	}
}

func TestLookupMiss(t *testing.T) {
	c := New()
	if _, ok := c.Lookup(net.ParseIP("1.2.3.4")); ok {
		t.Error("Lookup on empty cache should miss")
	}
}

func TestExpiredEntryIsAMiss(t *testing.T) {
	c := New()
	ip := net.ParseIP("1.2.3.4")
	c.mu.Lock()
	c.entries[ip.String()] = entry{domain: "stale.example", expiry: time.Now().Add(-time.Second)}
	c.mu.Unlock()

	if _, ok := c.Lookup(ip); ok {
		t.Error("expired entry should not be returned")
	}
}

func TestZeroTTLClampedToMinimum(t *testing.T) {
	c := New()
	ip := net.ParseIP("1.2.3.4")
	c.Set(ip, "example.com", 0)

	if _, ok := c.Lookup(ip); !ok {
		t.Error("a zero-TTL entry should still be immediately look-up-able (clamped to minTTL)")
	}
}
