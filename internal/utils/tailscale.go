// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package utils

import (
	"net/netip"
	"net/url"
	"strings"

	"tailscale.com/net/tsaddr"
)

// IsTailscaleIP checks if a given URL contains a Tailscale IP address
func IsTailscaleIP(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	ip, err := netip.ParseAddr(parsedURL.Hostname())
	if err != nil {
		return false
	}

	return tsaddr.IsTailscaleIP(ip)
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
