package utils

import (
	"net"
	"net/url"
	"strings"
)

// IsTailscaleIP checks if a given URL contains a Tailscale IP address
func IsTailscaleIP(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	if host == "" {
		return false
	}

	// Parse the IP address
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check if it's in the Tailscale CGNAT range (100.64.0.0/10)
	_, tailscaleNet, _ := net.ParseCIDR("100.64.0.0/10")
	if tailscaleNet != nil && tailscaleNet.Contains(ip) {
		return true
	}

	// Check if it's in the Tailscale IPv6 range (fd7a:115c:a1e0::/48)
	_, tailscaleNet6, _ := net.ParseCIDR("fd7a:115c:a1e0::/48")
	if tailscaleNet6 != nil && tailscaleNet6.Contains(ip) {
		return true
	}

	return false
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