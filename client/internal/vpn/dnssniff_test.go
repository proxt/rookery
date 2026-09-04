package vpn

import (
	"net"
	"testing"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/rookery/client/internal/vpn/dnscache"
)

// buildDNSResponse encodes a minimal, real DNS response for name -> ipv4,
// the same shape a stub resolver would receive back from a real query —
// this exercises SnoopDNSResponse against actual wire-format bytes, not a
// hand-rolled stand-in.
func buildDNSResponse(t *testing.T, name string, ipv4 net.IP, ttl uint32) []byte {
	t.Helper()

	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true})
	if err := b.StartQuestions(); err != nil {
		t.Fatalf("start questions: %v", err)
	}
	dnsName, err := dnsmessage.NewName(name)
	if err != nil {
		t.Fatalf("new name: %v", err)
	}
	if err := b.Question(dnsmessage.Question{Name: dnsName, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}); err != nil {
		t.Fatalf("add question: %v", err)
	}
	if err := b.StartAnswers(); err != nil {
		t.Fatalf("start answers: %v", err)
	}
	var addr [4]byte
	copy(addr[:], ipv4.To4())
	if err := b.AResource(
		dnsmessage.ResourceHeader{Name: dnsName, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: ttl},
		dnsmessage.AResource{A: addr},
	); err != nil {
		t.Fatalf("add A resource: %v", err)
	}
	data, err := b.Finish()
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	return data
}

func TestSnoopDNSResponsePopulatesCache(t *testing.T) {
	cache := dnscache.New()
	ip := net.ParseIP("93.184.216.34")
	data := buildDNSResponse(t, "example.com.", ip, 300)

	SnoopDNSResponse(cache, data)

	domain, ok := cache.Lookup(ip)
	if !ok {
		t.Fatalf("expected cache entry for %s", ip)
	}
	if domain != "example.com" {
		t.Errorf("domain = %q, want %q (no trailing dot)", domain, "example.com")
	}
}

func TestSnoopDNSResponseIgnoresQueries(t *testing.T) {
	cache := dnscache.New()

	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: false})
	b.StartQuestions()
	name, _ := dnsmessage.NewName("example.com.")
	b.Question(dnsmessage.Question{Name: name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET})
	data, err := b.Finish()
	if err != nil {
		t.Fatalf("finish: %v", err)
	}

	SnoopDNSResponse(cache, data)

	if _, ok := cache.Lookup(net.ParseIP("1.2.3.4")); ok {
		t.Error("a DNS query (not a response) should not populate the cache")
	}
}

func TestSnoopDNSResponseIgnoresGarbage(t *testing.T) {
	cache := dnscache.New()
	SnoopDNSResponse(cache, []byte("not a dns message"))
	if _, ok := cache.Lookup(net.ParseIP("1.2.3.4")); ok {
		t.Error("garbage payload should not populate the cache")
	}
}
