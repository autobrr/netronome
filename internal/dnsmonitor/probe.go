// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package dnsmonitor sends DNS queries to a resolver on a schedule and records
// the response time and the response code. It measures the resolver, not the
// record: it does not check that an answer is correct.
package dnsmonitor

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/autobrr/netronome/internal/types"
)

// queryTimeout is the time limit for one DNS query.
const queryTimeout = 5 * time.Second

// Default ports per protocol.
const (
	defaultPortPlain = "53"
	defaultPortDoT   = "853"
)

// Check is the outcome of one DNS query.
type Check struct {
	ResponseTime time.Duration
	ResponseCode string
	Success      bool
	Err          error
}

// RecordTypes are the record types a DNS monitor can ask for.
var RecordTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT"}

// Probe sends one query to the monitor's resolver. A timeout, a transport
// error, and an error response code (SERVFAIL, REFUSED, NXDOMAIN, and the like)
// all count as a failure.
func Probe(monitor *types.DNSMonitor) Check {
	recordType, ok := dns.StringToType[strings.ToUpper(monitor.RecordType)]
	if !ok {
		return Check{Err: fmt.Errorf("unknown record type %q", monitor.RecordType)}
	}

	client := &dns.Client{Net: network(monitor.Protocol), Timeout: queryTimeout}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(monitor.Query), recordType)

	// With a nil TLSConfig, DoT verifies the certificate against the host part
	// of the address. A mismatch fails the check, which is the correct signal.
	response, rtt, err := client.Exchange(msg, address(monitor.Host, monitor.Protocol))
	if err != nil {
		return Check{ResponseTime: rtt, Err: err}
	}

	code := dns.RcodeToString[response.Rcode]
	if response.Rcode != dns.RcodeSuccess {
		return Check{ResponseTime: rtt, ResponseCode: code, Err: fmt.Errorf("response code %s", code)}
	}

	return Check{ResponseTime: rtt, ResponseCode: code, Success: true}
}

// address adds the default port for the protocol when the host has none.
func address(host, protocol string) string {
	host = strings.TrimSpace(host)
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}

	port := defaultPortPlain
	if protocol == types.DNSProtocolDoT {
		port = defaultPortDoT
	}

	return net.JoinHostPort(host, port)
}

func network(protocol string) string {
	switch protocol {
	case types.DNSProtocolTCP:
		return "tcp"
	case types.DNSProtocolDoT:
		return "tcp-tls"
	default:
		return "udp"
	}
}
