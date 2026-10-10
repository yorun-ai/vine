// Package listenip defines Portal's explicit and legacy listener address semantics.
package listenip

import (
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
)

// Normalize returns sorted, unique IP literals. An empty list preserves the
// legacy wildcard TCP listener; explicit addresses bind only their IP family.
func Normalize(values []string) ([]string, error) {
	ips := make([]string, 0, len(values))
	for _, value := range values {
		addr, err := netip.ParseAddr(strings.TrimSpace(value))
		if err != nil || addr.Zone() != "" {
			return nil, fmt.Errorf("listenIPs must contain IP addresses without ports or zones: %q", value)
		}
		ips = append(ips, addr.Unmap().String())
	}
	slices.Sort(ips)
	ips = slices.Compact(ips)
	for i, left := range ips {
		for _, right := range ips[i+1:] {
			if Overlap(left, right) {
				return nil, fmt.Errorf("listenIPs contains overlapping addresses %q and %q", left, right)
			}
		}
	}
	return ips, nil
}

// Addresses expands the default configuration to the legacy listener sentinel.
func Addresses(ips []string) []string {
	if len(ips) == 0 {
		return []string{""}
	}
	return ips
}

// Binding returns the network and socket address for one normalized IP.
func Binding(ip string, port int) (string, string) {
	network := "tcp"
	if ip == "" {
		ip = "0.0.0.0"
	} else if netip.MustParseAddr(ip).Is4() {
		network = "tcp4"
	} else {
		network = "tcp6"
	}
	return network, net.JoinHostPort(ip, strconv.Itoa(port))
}

// Overlap reports whether two listeners may bind the same socket address. The
// legacy wildcard is conservatively treated as covering both address families.
func Overlap(left string, right string) bool {
	if left == "" || right == "" || left == right {
		return true
	}
	a := netip.MustParseAddr(left)
	b := netip.MustParseAddr(right)
	return a.Is4() == b.Is4() && (a.IsUnspecified() || b.IsUnspecified())
}
