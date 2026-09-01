// Package vpn captures all of the OS's IP traffic through a virtual network
// adapter (Wintun) and relays each TCP/UDP flow through the tunnel, the same
// way engine's SOCKS5 listener does for individual apps — this is what
// backs the GUI's "route everything" mode instead of per-app SOCKS5.
package vpn

import (
	"fmt"

	wgtun "golang.zx2c4.com/wireguard/tun"
)

// InterfaceName is the Wintun adapter's name, as it appears in Windows
// network settings.
const InterfaceName = "RookeryTun"

// MTU matches wireguard-go's own default; conservative enough to avoid
// fragmentation on typical paths.
const MTU = 1420

// InterfaceIP is the address assigned to the Wintun adapter. It's picked
// from the CGNAT range (100.64.0.0/10) specifically because home/office
// LANs essentially never use it, so collisions are unlikely.
const InterfaceIP = "100.100.0.2"

// InterfaceMask is InterfaceIP's subnet mask.
const InterfaceMask = "255.255.255.0"

// DNSServer is pushed to the adapter so DNS lookups also go through the
// tunnel instead of leaking to whatever the OS would otherwise use.
const DNSServer = "1.1.1.1"

// OpenDevice creates (or reuses, if one with the same name already exists
// from an unclean previous shutdown) the Wintun adapter.
func OpenDevice() (wgtun.Device, error) {
	dev, err := wgtun.CreateTUN(InterfaceName, MTU)
	if err != nil {
		return nil, fmt.Errorf("vpn: create wintun adapter: %w", err)
	}
	return dev, nil
}
