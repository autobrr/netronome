// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tailscale

import (
	"context"
	"fmt"
	"net"
	"time"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// Client provides a unified interface for both tsnet and host tailscaled
type Client interface {
	Status(ctx context.Context) (*ipnstate.Status, error)
}

// Mode represents how we're connecting to Tailscale
type Mode string

const (
	ModeHost  Mode = "host"  // Using host's tailscaled
	ModeTsnet Mode = "tsnet" // Using embedded tsnet
)

// hostClient wraps the system tailscaled client
type hostClient struct {
	client *tailscale.LocalClient
}

func (h *hostClient) Status(ctx context.Context) (*ipnstate.Status, error) {
	return h.client.Status(ctx)
}

// tsnetClient wraps a tsnet server's local client
type tsnetClient struct {
	client *tailscale.LocalClient
}

func (t *tsnetClient) Status(ctx context.Context) (*ipnstate.Status, error) {
	return t.client.Status(ctx)
}

// GetHostClient attempts to connect to the host's tailscaled
func GetHostClient() (Client, error) {
	// Try default client first (it will auto-detect socket/HTTP)
	client := &tailscale.LocalClient{}
	
	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if _, err := client.Status(ctx); err == nil {
		return &hostClient{client: client}, nil
	}
	
	return nil, fmt.Errorf("no running tailscaled found on host")
}

// GetTsnetClient creates a client from a tsnet server
func GetTsnetClient(server *tsnet.Server) (Client, error) {
	localClient, err := server.LocalClient()
	if err != nil {
		return nil, err
	}
	return &tsnetClient{client: localClient}, nil
}

// ListenOnTailscale listens on the Tailscale network if available
func ListenOnTailscale(hostClient Client, port int) (net.Listener, error) {
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
func GetSelfInfo(client Client) (hostname string, ips []string, err error) {
	status, err := client.Status(context.Background())
	if err != nil {
		return "", nil, err
	}
	
	if status.Self == nil {
		return "", nil, fmt.Errorf("no self information available")
	}
	
	hostname = status.Self.HostName
	for _, ip := range status.Self.TailscaleIPs {
		ips = append(ips, ip.String())
	}
	
	return hostname, ips, nil
}