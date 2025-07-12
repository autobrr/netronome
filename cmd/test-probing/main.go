// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func main() {
	host := "google.com"
	if len(os.Args) > 1 {
		host = os.Args[1]
	}

	fmt.Printf("Testing pro-bing with host: %s\n", host)
	fmt.Println("=" + string(make([]byte, 50)) + "=")

	// Test both privileged and unprivileged modes
	modes := []struct {
		name       string
		privileged bool
	}{
		{"Unprivileged", false},
		{"Privileged", true},
	}

	for _, mode := range modes {
		fmt.Printf("\n%s Mode Test:\n", mode.name)
		fmt.Println("-" + string(make([]byte, 30)) + "-")

		if err := runTest(host, mode.privileged); err != nil {
			log.Printf("Error in %s mode: %v\n", mode.name, err)
		}

		time.Sleep(2 * time.Second)
	}
}

func runTest(host string, privileged bool) error {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return fmt.Errorf("failed to create pinger: %w", err)
	}

	// Configure pinger
	pinger.Count = 10
	pinger.Interval = 1 * time.Second
	pinger.Timeout = 15 * time.Second
	pinger.SetPrivileged(privileged)

	// Track packets
	var packetsSent int32
	var packetsReceived int32
	var duplicates int32

	// Set up callbacks
	pinger.OnSend = func(pkt *probing.Packet) {
		atomic.AddInt32(&packetsSent, 1)
		fmt.Printf("[SEND] Packet #%d sent to %s\n", pkt.Seq, pkt.IPAddr)
	}

	pinger.OnRecv = func(pkt *probing.Packet) {
		atomic.AddInt32(&packetsReceived, 1)
		sent := atomic.LoadInt32(&packetsSent)
		recv := atomic.LoadInt32(&packetsReceived)
		progress := float64(recv) / float64(pinger.Count) * 100

		fmt.Printf("[RECV] Packet #%d from %s: RTT=%v, TTL=%d (Progress: %.1f%%, Sent: %d, Recv: %d)\n",
			pkt.Seq, pkt.IPAddr, pkt.Rtt, pkt.TTL, progress, sent, recv)
	}

	pinger.OnDuplicateRecv = func(pkt *probing.Packet) {
		atomic.AddInt32(&duplicates, 1)
		fmt.Printf("[DUP]  Duplicate packet #%d from %s\n", pkt.Seq, pkt.IPAddr)
	}

	pinger.OnFinish = func(stats *probing.Statistics) {
		fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		fmt.Printf("Packets: Sent = %d, Received = %d, Lost = %d (%.1f%% loss), Duplicates = %d\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketsSent-stats.PacketsRecv,
			stats.PacketLoss, stats.PacketsRecvDuplicates)
		fmt.Printf("Round-trip: Min = %v, Max = %v, Avg = %v, StdDev = %v\n",
			stats.MinRtt, stats.MaxRtt, stats.AvgRtt, stats.StdDevRtt)

		// Compare with our tracking
		sent := atomic.LoadInt32(&packetsSent)
		recv := atomic.LoadInt32(&packetsReceived)
		dup := atomic.LoadInt32(&duplicates)
		fmt.Printf("\nCallback tracking: Sent = %d, Received = %d, Duplicates = %d\n", sent, recv, dup)
	}

	// Run the pinger
	fmt.Printf("Starting ping test (privileged=%v)...\n", privileged)
	err = pinger.Run()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}
