package routing

import (
	_ "embed"
	"encoding/binary"
	"net"
	"sort"
)

// countriesIPv4 is a compact, sorted table of (range-start, country-code)
// pairs covering the whole IPv4 space, generated from the free (CC-BY 4.0,
// attribution in About.svelte and docs) DB-IP "country lite" dataset —
// https://db-ip.com — via a one-off converter, not parsed at runtime. Each
// record is 6 bytes: 4-byte big-endian range start + 2-byte ASCII country
// code. A range's end is implicit (the next record's start - 1), so the
// table only needs to be binary-searched, not interval-tree-matched.
//
// IPv6 is not covered — the source dataset's IPv6 table is ~10x larger for
// comparatively little traffic today, and this scope cut is worth taking
// explicitly rather than silently mis-embedding a much bigger binary. A
// GeoIP rule on an IPv6 destination simply never matches (falls through to
// whatever the next rule/default decides).
//
//go:embed geoipdata/countries_ipv4.bin
var countriesIPv4 []byte

const geoipRecordSize = 6

func geoipRecordCount() int {
	return len(countriesIPv4) / geoipRecordSize
}

func geoipStartAt(i int) uint32 {
	off := i * geoipRecordSize
	return binary.BigEndian.Uint32(countriesIPv4[off : off+4])
}

func geoipCodeAt(i int) string {
	off := i*geoipRecordSize + 4
	return string(countriesIPv4[off : off+2])
}

// countryFor returns the two-letter country code the embedded table
// attributes ip to, or "" if ip isn't in the table (not an IPv4 address, or
// before the first covered range).
func countryFor(ip net.IP) string {
	v4 := ip.To4()
	if v4 == nil {
		return ""
	}
	target := binary.BigEndian.Uint32(v4)

	n := geoipRecordCount()
	// Last record whose start is <= target.
	i := sort.Search(n, func(i int) bool { return geoipStartAt(i) > target }) - 1
	if i < 0 {
		return ""
	}
	return geoipCodeAt(i)
}
