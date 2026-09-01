package vpn

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// Windows can query DNS servers on every active network adapter at once and
// use whichever answers first ("smart multi-homed name resolution") —
// meaning name lookups can keep leaking to the original (non-tunnel) DNS
// server even after the tunnel's adapter becomes the preferred route. This
// is the standard fix VPN clients use: the same registry value Group
// Policy's "Turn off smart multi-homed name resolution" sets.
const (
	dnsPolicyKeyPath = `SOFTWARE\Policies\Microsoft\Windows NT\DNSClient`
	dnsPolicyValue   = "DisableSmartNameResolution"
)

// disableSmartMultiHomedResolution sets the policy, returning whether it
// changed anything (so Teardown only reverts what Setup actually touched).
func disableSmartMultiHomedResolution() (changed bool, err error) {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, dnsPolicyKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("vpn: open dns policy key: %w", err)
	}
	defer key.Close()

	existing, _, err := key.GetIntegerValue(dnsPolicyValue)
	if err == nil && existing == 1 {
		return false, nil // already disabled by something else; leave it alone on teardown
	}

	if err := key.SetDWordValue(dnsPolicyValue, 1); err != nil {
		return false, fmt.Errorf("vpn: set dns policy: %w", err)
	}
	return true, nil
}

func restoreSmartMultiHomedResolution() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, dnsPolicyKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("vpn: open dns policy key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(dnsPolicyValue); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("vpn: remove dns policy: %w", err)
	}
	return nil
}
