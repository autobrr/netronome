// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationTitle(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{"", "Netronome"},
		{"speedtest", "Netronome: Speedtest"},
		{"packetloss", "Netronome: Packet Loss"},
		{"agent", "Netronome: Agent"},
	}

	for _, tt := range tests {
		if got := notificationTitle(tt.category); got != tt.want {
			t.Errorf("notificationTitle(%q) = %q, want %q", tt.category, got, tt.want)
		}
	}
}

func TestSendDirectSetsTitle(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	url := "generic://" + strings.TrimPrefix(srv.URL, "http://") + "?template=json&disabletls=yes"
	notifier, err := NewNotifierFromURLs([]string{url})
	if err != nil {
		t.Fatalf("NewNotifierFromURLs: %v", err)
	}

	if err := notifier.sendDirect("hello"); err != nil {
		t.Fatalf("sendDirect: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal payload %q: %v", body, err)
	}
	if payload["title"] != "Netronome" {
		t.Errorf("payload title = %q, want %q", payload["title"], "Netronome")
	}
	if payload["message"] != "hello" {
		t.Errorf("payload message = %q, want %q", payload["message"], "hello")
	}
}
