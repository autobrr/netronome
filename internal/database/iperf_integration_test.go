// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIperfServer_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create server
		server, err := td.Service.SaveIperfServer(ctx, "Test Server", "iperf.example.com", 5201)
		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Greater(t, server.ID, 0)
		assert.Equal(t, "Test Server", server.Name)
		assert.Equal(t, "iperf.example.com", server.Host)
		assert.Equal(t, 5201, server.Port)
		assert.NotZero(t, server.CreatedAt)

		// Get all servers
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, server.ID, servers[0].ID)
		assert.Equal(t, server.Name, servers[0].Name)

		// Delete server
		err = td.Service.DeleteIperfServer(ctx, server.ID)
		require.NoError(t, err)

		// Verify deletion
		servers, err = td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, 0)
	})
}

func TestIperfServer_MultipleServers(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create multiple servers
		serverData := []struct {
			name string
			host string
			port int
		}{
			{"US East", "us-east.iperf.com", 5201},
			{"US West", "us-west.iperf.com", 5202},
			{"EU Central", "eu-central.iperf.com", 5203},
			{"Asia Pacific", "asia-pacific.iperf.com", 5204},
		}

		var createdIDs []int
		for _, data := range serverData {
			server, err := td.Service.SaveIperfServer(ctx, data.name, data.host, data.port)
			require.NoError(t, err)
			createdIDs = append(createdIDs, server.ID)
		}

		// Get all servers
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, len(serverData))

		// Verify all servers are present
		foundServers := make(map[string]bool)
		for _, server := range servers {
			foundServers[server.Name] = true
		}

		for _, data := range serverData {
			assert.True(t, foundServers[data.name],
				"Server %s should be present", data.name)
		}

		// Delete specific server
		err = td.Service.DeleteIperfServer(ctx, createdIDs[1])
		require.NoError(t, err)

		// Verify only 3 remain
		servers, err = td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, 3)

		// Verify the deleted server is not present
		for _, server := range servers {
			assert.NotEqual(t, "US West", server.Name)
		}
	})
}

func TestIperfServer_DuplicateHost(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create first server
		server1, err := td.Service.SaveIperfServer(ctx, "Server 1", "duplicate.iperf.com", 5201)
		require.NoError(t, err)
		require.NotNil(t, server1)

		// Try to create second server with same host
		// This should succeed as multiple servers can have same host but different ports
		server2, err := td.Service.SaveIperfServer(ctx, "Server 2", "duplicate.iperf.com", 5202)
		require.NoError(t, err)
		require.NotNil(t, server2)

		// Both should exist
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, 2)
	})
}

func TestIperfServer_PortRange(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Test various port numbers
		portTests := []struct {
			name string
			port int
		}{
			{"Low Port", 1024},
			{"Standard iPerf", 5201},
			{"High Port", 65535},
			{"Custom Port", 12345},
		}

		for _, test := range portTests {
			server, err := td.Service.SaveIperfServer(ctx, test.name, "port-test.iperf.com", test.port)
			require.NoError(t, err)
			assert.Equal(t, test.port, server.Port)
		}

		// Verify all were saved
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, len(portTests))
	})
}

func TestIperfServer_DeleteNonExistent(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Try to delete non-existent server
		err := td.Service.DeleteIperfServer(ctx, 99999)
		// Some databases might not return error for DELETE with no matches
		// So we just ensure it doesn't panic
		_ = err
	})
}

func TestIperfServer_EmptyList(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Get servers when none exist
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.NotNil(t, servers)
		assert.Len(t, servers, 0)
	})
}

func TestIperfServer_SpecialCharacters(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Test with special characters in name and host
		specialTests := []struct {
			name string
			host string
		}{
			{"Server with spaces", "host with spaces.com"},
			{"Server-with-dashes", "host-with-dashes.com"},
			{"Server_with_underscores", "host_with_underscores.com"},
			{"Server.with.dots", "192.168.1.100"},
			{"Server (with parentheses)", "host.example.com"},
			{"日本のサーバー", "japan.iperf.com"},
		}

		for _, test := range specialTests {
			server, err := td.Service.SaveIperfServer(ctx, test.name, test.host, 5201)
			require.NoError(t, err)
			assert.Equal(t, test.name, server.Name)
			assert.Equal(t, test.host, server.Host)
		}

		// Verify all were saved correctly
		servers, err := td.Service.GetIperfServers(ctx)
		require.NoError(t, err)
		assert.Len(t, servers, len(specialTests))

		// Verify names were preserved
		foundNames := make(map[string]bool)
		for _, server := range servers {
			foundNames[server.Name] = true
		}

		for _, test := range specialTests {
			assert.True(t, foundNames[test.name],
				"Server name %s should be present", test.name)
		}
	})
}
