// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func TestServerAddress(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		defaultPort string
		wantHost    string
		wantPort    string
	}{
		{name: "empty address", address: "", defaultPort: "5201", wantHost: "", wantPort: "5201"},
		{name: "ipv4 only", address: "192.0.2.10", defaultPort: "5201", wantHost: "192.0.2.10", wantPort: "5201"},
		{name: "ipv4 with port", address: "192.0.2.10:5202", defaultPort: "5201", wantHost: "192.0.2.10", wantPort: "5202"},
		{name: "hostname only", address: "iperf.example.com", defaultPort: "5201", wantHost: "iperf.example.com", wantPort: "5201"},
		{name: "hostname with port", address: "iperf.example.com:5202", defaultPort: "5201", wantHost: "iperf.example.com", wantPort: "5202"},
		{name: "bare ipv6", address: "2001:db8::1", defaultPort: "5201", wantHost: "2001:db8::1", wantPort: "5201"},
		{name: "bracketed ipv6", address: "[2001:db8::1]", defaultPort: "5201", wantHost: "2001:db8::1", wantPort: "5201"},
		{name: "bracketed ipv6 with port", address: "[2001:db8::1]:5202", defaultPort: "5201", wantHost: "2001:db8::1", wantPort: "5202"},
		{name: "surrounding spaces", address: "  [2001:db8::1]:5202  ", defaultPort: "5201", wantHost: "2001:db8::1", wantPort: "5202"},
		{name: "empty port falls back", address: "iperf.example.com:", defaultPort: "5201", wantHost: "iperf.example.com", wantPort: "5201"},
		{name: "no default port", address: "2001:db8::1", defaultPort: "", wantHost: "2001:db8::1", wantPort: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := serverAddress(tt.address, tt.defaultPort)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantPort, port)
		})
	}
}

// TestRunSingleIperfTestKeepsIPv6AddressWhole replaces iperf3 with a script
// that records its arguments. A colon split cuts an IPv6 address down to its
// first group and passes the second group as the port.
func TestRunSingleIperfTestKeepsIPv6AddressWhole(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture is a POSIX shell script")
	}

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "iperf3-args")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" > '" + argsFile + "'\n" +
		"printf '%s\\n' '{\"end\":{\"sum_sent\":{\"bits_per_second\":100000000},\"sum_received\":{\"bits_per_second\":200000000}}}'\n"
	scriptPath := filepath.Join(dir, "iperf3")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	runner := NewIperfRunner(config.IperfConfig{TestDuration: 1, ParallelConns: 1, Timeout: 30})
	opts := &types.TestOptions{
		ServerHost:     "[2001:db8::1]:5203",
		EnableDownload: true,
	}

	result, err := runner.runSingleIperfTest(context.Background(), opts)
	require.NoError(t, err)
	assert.InDelta(t, 200.0, result.DownloadSpeed, 0.0001)

	rawArgs, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	argv := strings.Fields(strings.TrimSpace(string(rawArgs)))

	host, hostFound := flagValue(argv, "-c")
	require.True(t, hostFound, "iperf3 must receive -c, got %v", argv)
	assert.Equal(t, "2001:db8::1", host)

	port, portFound := flagValue(argv, "-p")
	require.True(t, portFound, "iperf3 must receive -p, got %v", argv)
	assert.Equal(t, "5203", port)
}

func flagValue(argv []string, flag string) (string, bool) {
	for i, arg := range argv {
		if arg == flag && i+1 < len(argv) {
			return argv[i+1], true
		}
	}
	return "", false
}
