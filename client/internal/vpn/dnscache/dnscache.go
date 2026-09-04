// Package dnscache maps a resolved IP address back to the domain name that
// was looked up for it. Under system-wide (TUN) capture, a later TCP/UDP
// flow only ever carries a destination IP — this cache is what lets a
// domain-based routing rule still apply, by reversing the DNS lookup that
// (almost always) preceded the connection.
package dnscache

import (
	"net"
	"sync"
	"time"
)

type entry struct {
	domain string
	expiry time.Time
}

// Cache is a small IP -> domain map with per-entry TTL, safe for concurrent
// use. The zero value is not usable — construct with New.
type Cache struct {
	mu      sync.Mutex
	entries map[string]entry
}

// New creates an empty Cache.
func New() *Cache {
	return &Cache{entries: make(map[string]entry)}
}

// Set records that ip was resolved for domain, valid for ttl. A ttl of zero
// or less is clamped to a floor so a malformed/zero DNS TTL doesn't expire
// the entry before it can ever be looked up.
func (c *Cache) Set(ip net.IP, domain string, ttl time.Duration) {
	if ip == nil || domain == "" {
		return
	}
	if ttl < minTTL {
		ttl = minTTL
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[ip.String()] = entry{domain: domain, expiry: time.Now().Add(ttl)}
}

// minTTL is the floor applied to Set's ttl — some resolvers/records use a
// 0 TTL, which would otherwise make the entry useless for the very next
// connection attempt (typically made within milliseconds of the lookup).
const minTTL = 5 * time.Second

// Lookup returns the domain last resolved to ip, if the entry hasn't
// expired.
func (c *Cache) Lookup(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[ip.String()]
	if !ok || time.Now().After(e.expiry) {
		return "", false
	}
	return e.domain, true
}
