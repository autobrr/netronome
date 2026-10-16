// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

func TestNormalizeDNSMonitor(t *testing.T) {
	tests := []struct {
		name    string
		input   types.DNSMonitor
		want    types.DNSMonitor
		wantErr string
	}{
		{
			name:  "fills in the defaults",
			input: types.DNSMonitor{Host: "1.1.1.1"},
			want:  types.DNSMonitor{Host: "1.1.1.1", Protocol: "udp", Query: "google.com", RecordType: "A", Interval: "60s"},
		},
		{
			name:  "trims the host and keeps the given values",
			input: types.DNSMonitor{Host: "  dns.example.invalid:853 ", Protocol: "DoT", Query: " home.arpa ", RecordType: "aaaa", Interval: "5m"},
			want:  types.DNSMonitor{Host: "dns.example.invalid:853", Protocol: "dot", Query: "home.arpa", RecordType: "AAAA", Interval: "5m"},
		},
		{
			name:    "rejects an empty host",
			input:   types.DNSMonitor{Host: "   "},
			wantErr: "Host is required",
		},
		{
			name:    "rejects an unknown protocol",
			input:   types.DNSMonitor{Host: "1.1.1.1", Protocol: "doh"},
			wantErr: "Protocol must be one of: udp, tcp, dot",
		},
		{
			name:    "rejects an unknown record type",
			input:   types.DNSMonitor{Host: "1.1.1.1", RecordType: "SOA"},
			wantErr: "Record type must be one of: A, AAAA, CNAME, MX, NS, TXT",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			monitor := test.input
			err := normalizeDNSMonitor(&monitor)

			if test.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, test.wantErr, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, monitor)
		})
	}
}
