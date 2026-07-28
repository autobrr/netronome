// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tailscale

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

// Client talks to a tailscaled LocalAPI, whether that is the host's daemon or
// an embedded tsnet server. Both hand back the same type, so there is nothing
// to abstract over.
type Client = local.Client

// Mode represents how we're connecting to Tailscale
type Mode string

const (
	ModeHost  Mode = "host"  // Using host's tailscaled
	ModeTsnet Mode = "tsnet" // Using embedded tsnet
)

// GetHostClient attempts to connect to the host's tailscaled
func GetHostClient() (*Client, error) {
	// Try default client first (it will auto-detect socket/HTTP)
	client := &Client{}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Status(ctx); err == nil {
		return client, nil
	}

	return nil, fmt.Errorf("no running tailscaled found on host")
}

// GetTsnetClient creates a client from a tsnet server
func GetTsnetClient(server *tsnet.Server) (*Client, error) {
	return server.LocalClient()
}

// ListenOnTailscale listens on the Tailscale network if available
func ListenOnTailscale(hostClient *Client, port int) (net.Listener, error) {
	status, err := hostClient.Status(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	if status.Self == nil || len(status.Self.TailscaleIPs) == 0 {
		return nil, fmt.Errorf("no Tailscale IPs available")
	}

	// Listen on the first Tailscale IP
	addr := fmt.Sprintf("%s:%d", status.Self.TailscaleIPs[0], port)
	return net.Listen("tcp", addr)
}

// GetSelfInfo returns information about the current Tailscale node
func GetSelfInfo(client *Client) (hostname string, ips []string, err error) {
	status, err := client.Status(context.Background())
	if err != nil {
		return "", nil, err
	}

	if status.Self == nil {
		return "", nil, fmt.Errorf("no self information available")
	}

	// Use the actual Tailscale machine name (DNSName without suffix)
	hostname = status.Self.DNSName
	// Trim the MagicDNS suffix to get just the machine name
	if hostname != "" && strings.Contains(hostname, ".") {
		hostname = strings.Split(hostname, ".")[0]
	}
	// Fallback to HostName if DNSName is empty
	if hostname == "" {
		hostname = status.Self.HostName
	}

	for _, ip := range status.Self.TailscaleIPs {
		ips = append(ips, ip.String())
	}

	return hostname, ips, nil
}
