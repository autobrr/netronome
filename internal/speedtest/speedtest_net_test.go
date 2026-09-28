// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	st "github.com/showwin/speedtest-go/speedtest"
	"github.com/stretchr/testify/assert"
)

func TestResolveServerURL(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"follows redirect", redirector.URL + "/speedtest/upload.php", target.URL + "/speedtest/upload.php"},
		{"keeps direct URL", target.URL + "/speedtest/upload.php", target.URL + "/speedtest/upload.php"},
		{"keeps URL on error", "http://127.0.0.1:1/speedtest/upload.php", "http://127.0.0.1:1/speedtest/upload.php"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &st.Server{URL: tt.url}
			resolveServerURL(t.Context(), server)
			assert.Equal(t, tt.want, server.URL)
		})
	}
}
