package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/netronome/internal/types"
)

func TestDetectAgentCapabilities_FromRootEndpoints(t *testing.T) {
	t.Run("has system+hardware", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"endpoints":{"system":"/system/info","hardware":"/system/hardware"}}`))
		}))
		t.Cleanup(srv.Close)

		caps, err := detectAgentCapabilities(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("detectAgentCapabilities error: %v", err)
		}
		if !caps.systemInfo.known || !caps.systemInfo.supported {
			t.Fatalf("systemInfo expected known+supported, got %+v", caps.systemInfo)
		}
		if !caps.hardwareStats.known || !caps.hardwareStats.supported {
			t.Fatalf("hardwareStats expected known+supported, got %+v", caps.hardwareStats)
		}
	})

	t.Run("missing system+hardware", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"endpoints":{"live":"/events?stream=live-data"}}`))
		}))
		t.Cleanup(srv.Close)

		caps, err := detectAgentCapabilities(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("detectAgentCapabilities error: %v", err)
		}
		if !caps.systemInfo.known || caps.systemInfo.supported {
			t.Fatalf("systemInfo expected known+unsupported, got %+v", caps.systemInfo)
		}
		if !caps.hardwareStats.known || caps.hardwareStats.supported {
			t.Fatalf("hardwareStats expected known+unsupported, got %+v", caps.hardwareStats)
		}
	})

	t.Run("missing endpoints field", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"service":"monitor SSE agent"}`))
		}))
		t.Cleanup(srv.Close)

		_, err := detectAgentCapabilities(context.Background(), srv.URL)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestClient_HandleEndpointNotFound_DisablesPolling(t *testing.T) {
	c := &Client{
		agent: &types.MonitorAgent{
			ID:  123,
			URL: "http://example.invalid/events?stream=live-data",
		},
	}

	if !c.shouldPollSystemInfo() {
		t.Fatalf("expected system info polling enabled by default")
	}

	if !c.handleEndpointNotFound(&httpStatusError{StatusCode: http.StatusNotFound, URL: "http://x/system/info"}, endpointSystemInfo) {
		t.Fatalf("expected handleEndpointNotFound=true")
	}
	if c.shouldPollSystemInfo() {
		t.Fatalf("expected system info polling disabled after 404")
	}
}
