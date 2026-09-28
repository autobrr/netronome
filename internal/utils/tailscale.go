// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package utils

import (
	"net/netip"
	"net/url"
	"strings"

	"tailscale.com/net/tsaddr"
)

// IsTailscaleIP reports whether the host of urlStr is in the Tailscale CGNAT or
// ULA range. Deliberately not tsaddr.IsTailscaleIP: that one subtracts the ChromeOS
// VM range 100.115.92.0/23, and agents on those addresses must still report as Tailscale.
func IsTailscaleIP(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	ip, err := netip.ParseAddr(parsedURL.Hostname())
	if err != nil {
		return false
	}

	// Unmap keeps ::ffff:100.64.0.1 matching the v4 range, as net.IP did.
	ip = ip.Unmap()
	return tsaddr.CGNATRange().Contains(ip) || tsaddr.TailscaleULARange().Contains(ip)
}

// IsTailscaleHostname checks if a hostname looks like a Tailscale MagicDNS name
func IsTailscaleHostname(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	if host == "" {
		return false
	}

	// Check for common Tailscale MagicDNS suffixes
	tailscaleSuffixes := []string{
		".ts.net",
		".beta.tailscale.net",
		".alpha.tailscale.net",
	}

	for _, suffix := range tailscaleSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}

// IsTailscaleURL checks if a URL is using Tailscale (either IP or hostname)
func IsTailscaleURL(urlStr string) bool {
	return IsTailscaleIP(urlStr) || IsTailscaleHostname(urlStr)
}
