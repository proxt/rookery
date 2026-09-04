package vpn

import (
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/rookery/client/internal/vpn/dnscache"
)

// SnoopDNSResponse parses payload as a DNS response and records every
// A/AAAA answer it contains in cache, keyed by the question name. Best
// effort by design: called on the hot path of every UDP:53 flow, so a
// malformed or non-DNS payload (this is only ever called for traffic to
// port 53, but nothing stops an app from sending garbage there) is silently
// ignored rather than surfaced — it just means routing falls back to
// GeoIP/default for that destination, not an error worth logging per packet.
//
// Only UDP DNS is covered — TCP DNS (large responses, zone transfers) is
// rare for a stub resolver's ordinary lookups and adds a length-prefix
// framing step for little practical benefit; a scope cut, not an oversight.
func SnoopDNSResponse(cache *dnscache.Cache, payload []byte) {
	var msg dnsmessage.Message
	if err := msg.Unpack(payload); err != nil || !msg.Header.Response {
		return
	}

	domain := ""
	if len(msg.Questions) > 0 {
		domain = normalizeDNSName(msg.Questions[0].Name.String())
	}
	if domain == "" {
		return
	}

	for _, a := range msg.Answers {
		ttl := time.Duration(a.Header.TTL) * time.Second
		switch body := a.Body.(type) {
		case *dnsmessage.AResource:
			cache.Set(net.IP(body.A[:]), domain, ttl)
		case *dnsmessage.AAAAResource:
			cache.Set(net.IP(body.AAAA[:]), domain, ttl)
		}
	}
}

// normalizeDNSName strips the trailing root dot dnsmessage.Name.String()
// always includes, matching the bare-hostname form routing.Matcher's domain
// rules are written against.
func normalizeDNSName(name string) string {
	if n := len(name); n > 0 && name[n-1] == '.' {
		return name[:n-1]
	}
	return name
}
