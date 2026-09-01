// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dnsmonitor

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

// answer replies with one A record for whatever was asked.
func answer(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	rr, err := dns.NewRR(req.Question[0].Name + " 60 IN A 127.0.0.1")
	if err == nil {
		msg.Answer = append(msg.Answer, rr)
	}
	_ = w.WriteMsg(msg)
}

func rcode(code int) dns.HandlerFunc {
	return func(w dns.ResponseWriter, req *dns.Msg) {
		msg := new(dns.Msg)
		msg.SetRcode(req, code)
		_ = w.WriteMsg(msg)
	}
}

// startUDPServer runs a DNS server on a loopback port and returns its address.
func startUDPServer(t *testing.T, handler dns.Handler) string {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &dns.Server{PacketConn: conn, Handler: handler}
	go func() { _ = server.ActivateAndServe() }()
	t.Cleanup(func() { _ = server.Shutdown() })

	return conn.LocalAddr().String()
}

func startTCPServer(t *testing.T, handler dns.Handler) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &dns.Server{Listener: listener, Handler: handler}
	go func() { _ = server.ActivateAndServe() }()
	t.Cleanup(func() { _ = server.Shutdown() })

	return listener.Addr().String()
}

func monitorFor(host, protocol string) *types.DNSMonitor {
	return &types.DNSMonitor{
		Host:       host,
		Protocol:   protocol,
		Query:      "example.invalid",
		RecordType: "A",
	}
}

func TestProbe_Success(t *testing.T) {
	addr := startUDPServer(t, dns.HandlerFunc(answer))

	check := Probe(monitorFor(addr, types.DNSProtocolUDP))

	assert.True(t, check.Success)
	assert.Equal(t, "NOERROR", check.ResponseCode)
	assert.NoError(t, check.Err)
	assert.Positive(t, check.ResponseTime)
}

func TestProbe_OverTCP(t *testing.T) {
	addr := startTCPServer(t, dns.HandlerFunc(answer))

	check := Probe(monitorFor(addr, types.DNSProtocolTCP))

	assert.True(t, check.Success)
	assert.Equal(t, "NOERROR", check.ResponseCode)
}

func TestProbe_ErrorResponseCodesFail(t *testing.T) {
	for name, code := range map[string]int{
		"SERVFAIL": dns.RcodeServerFailure,
		"REFUSED":  dns.RcodeRefused,
		"NXDOMAIN": dns.RcodeNameError,
	} {
		t.Run(name, func(t *testing.T) {
			addr := startUDPServer(t, rcode(code))

			check := Probe(monitorFor(addr, types.DNSProtocolUDP))

			assert.False(t, check.Success)
			assert.Equal(t, name, check.ResponseCode)
			assert.Error(t, check.Err)
		})
	}
}

func TestProbe_TimeoutFails(t *testing.T) {
	// a server that never answers, so the query runs into queryTimeout
	addr := startUDPServer(t, dns.HandlerFunc(func(dns.ResponseWriter, *dns.Msg) {}))

	check := Probe(monitorFor(addr, types.DNSProtocolUDP))

	assert.False(t, check.Success)
	assert.Empty(t, check.ResponseCode)
	require.Error(t, check.Err)
	assert.Contains(t, check.Err.Error(), "timeout")
}

func TestProbe_UnknownRecordType(t *testing.T) {
	monitor := monitorFor("127.0.0.1:5353", types.DNSProtocolUDP)
	monitor.RecordType = "BOGUS"

	check := Probe(monitor)

	assert.False(t, check.Success)
	require.Error(t, check.Err)
	assert.Contains(t, check.Err.Error(), "unknown record type")
}

func TestAddress(t *testing.T) {
	tests := []struct {
		host     string
		protocol string
		want     string
	}{
		{"1.1.1.1", types.DNSProtocolUDP, "1.1.1.1:53"},
		{"1.1.1.1", types.DNSProtocolTCP, "1.1.1.1:53"},
		{"one.one.one.one", types.DNSProtocolDoT, "one.one.one.one:853"},
		{"1.1.1.1:5353", types.DNSProtocolUDP, "1.1.1.1:5353"},
		{" 1.1.1.1 ", types.DNSProtocolUDP, "1.1.1.1:53"},
		{"2606:4700:4700::1111", types.DNSProtocolUDP, "[2606:4700:4700::1111]:53"},
		{"[2606:4700:4700::1111]:5353", types.DNSProtocolUDP, "[2606:4700:4700::1111]:5353"},
	}

	for _, test := range tests {
		t.Run(test.host+"/"+test.protocol, func(t *testing.T) {
			assert.Equal(t, test.want, address(test.host, test.protocol))
		})
	}
}
