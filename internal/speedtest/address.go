// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"net"
	"net/netip"
	"strings"
)

// serverAddress splits a configured server address into the host and port that
// an external tool needs. It accepts "host", "host:port", "1.2.3.4",
// "[::1]" and "[::1]:5201". A bare IPv6 address keeps its colons and gets
// defaultPort, because every colon belongs to the address.
func serverAddress(rawAddress, defaultPort string) (string, string) {
	address := strings.TrimSpace(rawAddress)
	if address == "" {
		return "", defaultPort
	}

	// SplitHostPort accepts "host:port" and "[::1]:port". It rejects a bare
	// IPv6 address, which has more than one colon.
	if host, port, err := net.SplitHostPort(address); err == nil {
		if port == "" {
			port = defaultPort
		}
		return host, port
	}

	// A bracketed IPv6 address without a port, such as "[::1]". netip keeps
	// the zone of a link-local address, which net.ParseIP rejects.
	if strings.HasPrefix(address, "[") && strings.HasSuffix(address, "]") {
		if addr, err := netip.ParseAddr(address[1 : len(address)-1]); err == nil {
			return addr.String(), defaultPort
		}
	}

	return address, defaultPort
}
