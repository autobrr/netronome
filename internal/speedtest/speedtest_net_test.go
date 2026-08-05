// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"errors"
	"testing"
	"time"

	st "github.com/showwin/speedtest-go/speedtest"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// A caller whose deadline passes while it waits for a running test must give up
// instead of starting a test nobody is waiting for any more.
func TestRunTestGivesUpWhenQueuedBehindARunningTest(t *testing.T) {
	r := NewSpeedtestNetRunner(config.SpeedTestConfig{Timeout: 60})
	r.running <- struct{}{} // another test holds the slot

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := r.RunTest(ctx, &types.TestOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

// Nothing is running, so taking the slot and observing the cancellation are both
// ready when the select is evaluated, and Go then picks one at random. Taking
// the slot must not be enough to start the test. No single run can force the
// random choice, so repeat: without the recheck about half of these reach the
// network and return something other than context.Canceled.
func TestRunTestGivesUpWhenSlotIsFree(t *testing.T) {
	for i := range 20 {
		r := NewSpeedtestNetRunner(config.SpeedTestConfig{Timeout: 60})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		errc := make(chan error, 1)
		go func() {
			_, err := r.RunTest(ctx, &types.TestOptions{})
			errc <- err
		}()

		select {
		case err := <-errc:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("run %d: got %v, want context.Canceled", i, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("run %d: RunTest started a test for a caller that had given up", i)
		}
	}
}

func TestSelectNearestServer(t *testing.T) {
	server := func(name string, distance float64, latency time.Duration) *st.Server {
		return &st.Server{Name: name, Distance: distance, Latency: latency}
	}

	tests := []struct {
		name    string
		servers []*st.Server
		want    string
	}{
		{
			name:    "no servers",
			servers: nil,
			want:    "",
		},
		{
			name:    "single server",
			servers: []*st.Server{server("only", 10, 20*time.Millisecond)},
			want:    "only",
		},
		{
			name: "closest wins when distances differ",
			servers: []*st.Server{
				server("far", 100, 1*time.Millisecond),
				server("near", 10, 50*time.Millisecond),
			},
			want: "near",
		},
		{
			name: "equal distance is broken on latency",
			servers: []*st.Server{
				server("slow", 23.5, 15*time.Millisecond),
				server("fast", 23.5, 8*time.Millisecond),
				server("mid", 23.5, 12*time.Millisecond),
			},
			want: "fast",
		},
		{
			name: "failed ping never wins",
			servers: []*st.Server{
				server("unreachable", 23.5, st.PingTimeout),
				server("reachable", 23.5, 30*time.Millisecond),
			},
			want: "reachable",
		},
		{
			name: "all pings failed falls back to the closest",
			servers: []*st.Server{
				server("closest", 10, st.PingTimeout),
				server("further", 20, st.PingTimeout),
			},
			want: "closest",
		},
		{
			name: "a distant server never wins on latency",
			servers: []*st.Server{
				server("c1", 1, 20*time.Millisecond),
				server("c2", 2, 20*time.Millisecond),
				server("c3", 3, 20*time.Millisecond),
				server("c4", 4, 20*time.Millisecond),
				server("c5", 5, 20*time.Millisecond),
				server("distant-but-quick", 900, 1*time.Millisecond),
			},
			want: "c1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectNearestServer(tt.servers)
			if tt.want == "" {
				if got != nil {
					t.Fatalf("got %q, want nil", got.Name)
				}
				return
			}
			if got == nil {
				t.Fatalf("got nil, want %q", tt.want)
			}
			if got.Name != tt.want {
				t.Errorf("got %q, want %q", got.Name, tt.want)
			}
		})
	}
}
