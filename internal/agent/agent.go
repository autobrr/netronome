// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
	"tailscale.com/tsnet"

	"github.com/autobrr/netronome/internal/config"
	ts "github.com/autobrr/netronome/internal/tailscale"
)


// New creates a new Agent instance
func New(cfg *config.AgentConfig) *Agent {
	return &Agent{
		config:      cfg,
		clients:     make(map[chan string]bool),
		monitorData: make(chan string, 100),
	}
}

// NewWithTailscale creates a new Agent instance with Tailscale support
func NewWithTailscale(cfg *config.AgentConfig, tsCfg *config.TailscaleConfig) *Agent {
	return &Agent{
		config:          cfg,
		tailscaleConfig: tsCfg,
		clients:         make(map[chan string]bool),
		monitorData:     make(chan string, 100),
		useTailscale:    tsCfg != nil && tsCfg.Enabled && tsCfg.Agent.Enabled,
	}
}

// Start starts the agent server
func (a *Agent) Start(ctx context.Context) error {
	// If Tailscale is enabled, use Tailscale start
	if a.useTailscale {
		// Try to use host's tailscaled first if configured
		if a.tailscaleConfig != nil && a.tailscaleConfig.PreferHost {
			log.Info().Msg("Attempting to use host's tailscaled...")
			if err := a.startWithHostTailscale(ctx); err != nil {
				log.Warn().Err(err).Msg("Failed to use host's tailscaled, falling back to tsnet")
			} else {
				return nil
			}
		}
		return a.startWithTailscale(ctx)
	}

	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)

	// Start broadcaster
	go a.broadcaster(ctx)

	// Set up routes
	router := a.setupRoutes()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
	}()

	if a.config.APIKey != "" {
		log.Info().Str("addr", addr).Msg("Starting monitor SSE agent with API key authentication")
	} else {
		log.Info().Str("addr", addr).Msg("Starting monitor SSE agent without authentication")
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// startWithTailscale starts the agent server with Tailscale
func (a *Agent) startWithTailscale(ctx context.Context) error {
	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)

	// Start broadcaster
	go a.broadcaster(ctx)

	// Configure tsnet
	hostname := a.tailscaleConfig.Hostname
	if hostname == "" {
		// Generate default hostname
		sysHostname, _ := os.Hostname()
		hostname = fmt.Sprintf("netronome-agent-%s", sysHostname)
	}

	// Expand state directory path
	stateDir := a.tailscaleConfig.StateDir
	if strings.HasPrefix(stateDir, "~/") {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, stateDir[2:])
	}

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create tsnet state directory: %w", err)
	}

	a.tsnetServer = &tsnet.Server{
		Dir:       stateDir,
		Hostname:  hostname,
		AuthKey:   a.tailscaleConfig.AuthKey,
		Ephemeral: a.tailscaleConfig.Ephemeral,
		Logf:      func(format string, args ...interface{}) {
			log.Debug().Msgf("[tsnet] "+format, args...)
		},
	}

	// Set control URL if specified
	if a.tailscaleConfig.ControlURL != "" {
		a.tsnetServer.ControlURL = a.tailscaleConfig.ControlURL
	}

	// Start tsnet server
	log.Info().Str("hostname", hostname).Msg("Starting Tailscale node...")
	if err := a.tsnetServer.Start(); err != nil {
		return fmt.Errorf("failed to start tsnet: %w", err)
	}

	// Set up routes
	router := a.setupRoutes()

	// Listen on Tailscale network
	ln, err := a.tsnetServer.Listen("tcp", fmt.Sprintf(":%d", a.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on Tailscale: %w", err)
	}

	server := &http.Server{
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down Tailscale agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
		if a.tsnetServer != nil {
			a.tsnetServer.Close()
		}
	}()

	localClient, err := a.tsnetServer.LocalClient()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get Tailscale local client")
	} else {
		status, err := localClient.Status(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get Tailscale status")
		} else if status.Self != nil {
			logEvent := log.Info().
				Str("hostname", hostname).
				Int("port", a.config.Port)
			
			if len(status.Self.TailscaleIPs) > 0 {
				logEvent = logEvent.Str("tailscale_ip", status.Self.TailscaleIPs[0].String())
			}
			
			logEvent.Msg("Monitor SSE agent listening on Tailscale network")
			
			if a.config.APIKey != "" {
				log.Info().Msg("API key authentication enabled")
			}
		}
	}

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// startWithHostTailscale starts the agent using the host's existing tailscaled
func (a *Agent) startWithHostTailscale(ctx context.Context) error {
	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)
	// Start broadcaster
	go a.broadcaster(ctx)

	// Get host tailscale client
	hostClient, err := ts.GetHostClient()
	if err != nil {
		return fmt.Errorf("failed to connect to host tailscaled: %w", err)
	}

	// Get our tailscale status
	hostname, ips, err := ts.GetSelfInfo(hostClient)
	if err != nil {
		return fmt.Errorf("failed to get Tailscale info: %w", err)
	}

	log.Info().
		Str("hostname", hostname).
		Strs("tailscale_ips", ips).
		Msg("Using host's tailscaled for agent")

	// Start normal server but bind to Tailscale IP
	router := a.setupRoutes()
	server := &http.Server{
		Handler: router,
	}

	// Listen on Tailscale network
	ln, err := ts.ListenOnTailscale(hostClient, a.config.Port)
	if err != nil {
		// Fallback to all interfaces if we can't bind to specific Tailscale IP
		log.Warn().Err(err).Msg("Failed to bind to Tailscale IP, using all interfaces")
		addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}

	// Set up graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
	}()

	log.Info().
		Str("address", ln.Addr().String()).
		Int("port", a.config.Port).
		Msg("Monitor SSE agent listening via host's tailscaled")

	if a.config.APIKey != "" {
		log.Info().Msg("API key authentication enabled")
	}

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// GetTailscaleStatus returns the current Tailscale connection status
func (a *Agent) GetTailscaleStatus() (map[string]interface{}, error) {
	if !a.useTailscale || a.tsnetServer == nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "disabled",
		}, nil
	}

	localClient, err := a.tsnetServer.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale local client: %w", err)
	}

	status, err := localClient.Status(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	result := map[string]interface{}{
		"enabled":  true,
		"hostname": a.tsnetServer.Hostname,
	}

	if status.Self != nil {
		var ips []string
		for _, ip := range status.Self.TailscaleIPs {
			ips = append(ips, ip.String())
		}
		result["tailscale_ips"] = ips
		result["online"] = status.Self.Online
		result["status"] = "connected"
	} else {
		result["status"] = "connecting"
	}

	return result, nil
}

// handleSSE handles SSE connections
func (a *Agent) handleSSE(c *gin.Context) {
	stream := c.Query("stream")
	if stream != "live-data" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream parameter"})
		return
	}

	// Create a client channel
	clientChan := make(chan string, 100)

	// Register client
	a.clientsMu.Lock()
	a.clients[clientChan] = true
	a.clientsMu.Unlock()

	// Clean up on disconnect
	defer func() {
		a.clientsMu.Lock()
		delete(a.clients, clientChan)
		a.clientsMu.Unlock()
		close(clientChan)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case data := <-clientChan:
			c.SSEvent("message", data)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// broadcaster distributes data to all connected clients
func (a *Agent) broadcaster(ctx context.Context) {
	for {
		select {
		case data := <-a.monitorData:
			a.clientsMu.RLock()
			for client := range a.clients {
				select {
				case client <- data:
				default:
					// Client buffer full, skip
				}
			}
			a.clientsMu.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}

// handleHistoricalExport exports all historical bandwidth monitor data
func (a *Agent) handleHistoricalExport(c *gin.Context) {
	// Get optional interface parameter
	iface := c.Query("interface")
	if iface == "" {
		iface = a.config.Interface
	}

	// Build vnstat command for all historical data
	args := []string{"--json", "a"}
	if iface != "" {
		args = append(args, "--iface", iface)
	}

	// Execute vnstat command
	cmd := exec.Command("vnstat", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("Failed to export historical data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to export historical data",
			"details": err.Error(),
		})
		return
	}

	// Parse the JSON to add timezone information
	var bandwidthData map[string]interface{}
	if err := json.Unmarshal(output, &bandwidthData); err != nil {
		log.Error().Err(err).Msg("Failed to parse bandwidth data JSON")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse bandwidth data",
		})
		return
	}

	// Add server time information for timezone handling
	now := time.Now()
	bandwidthData["server_time"] = now.Format(time.RFC3339)
	bandwidthData["server_time_unix"] = now.Unix()
	_, offset := now.Zone()
	bandwidthData["timezone_offset"] = offset // Offset in seconds from UTC

	// Re-encode with timezone information
	enrichedOutput, err := json.Marshal(bandwidthData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode bandwidth data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to encode data",
		})
		return
	}

	// Set appropriate headers for JSON response
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "inline; filename=\"monitor-historical.json\"")

	// Return the enriched JSON data
	c.Data(http.StatusOK, "application/json", enrichedOutput)
}

// runBandwidthMonitor runs vnstat command and sends data to the broadcast channel
func (a *Agent) runBandwidthMonitor(ctx context.Context) {
	// Build vnstat command
	args := []string{"--live", "--json"}
	if a.config.Interface != "" {
		args = append(args, "--iface", a.config.Interface)
	}

	cmd := exec.CommandContext(ctx, "vnstat", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create stdout pipe")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start vnstat")
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse JSON to validate it
		var data MonitorLiveData
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			log.Warn().Err(err).Str("line", line).Msg("Failed to parse bandwidth data JSON")
			continue
		}

		// Track peak speeds
		a.peakMu.Lock()
		if data.Rx.Bytespersecond > a.peakRx {
			a.peakRx = data.Rx.Bytespersecond
		}
		if data.Tx.Bytespersecond > a.peakTx {
			a.peakTx = data.Tx.Bytespersecond
		}
		a.peakMu.Unlock()

		// Send to broadcaster
		select {
		case a.monitorData <- line:
		default:
			// Channel full, skip
		}

		log.Trace().
			Str("rx", data.Rx.Ratestring).
			Str("tx", data.Tx.Ratestring).
			Msg("Broadcasting bandwidth monitor data")
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Scanner error")
	}

	if err := cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("vnstat command failed")
	}
}

// handleSystemInfo returns system and interface information
func (a *Agent) handleSystemInfo(c *gin.Context) {
	info, err := a.getSystemInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get system info",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// getSystemInfo gathers system and interface information
func (a *Agent) getSystemInfo() (*SystemInfo, error) {
	log.Debug().Msg("Starting getSystemInfo")

	info := &SystemInfo{
		Interfaces: make(map[string]InterfaceInfo),
		UpdatedAt:  time.Now(),
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
		log.Debug().Str("hostname", hostname).Msg("Got hostname")
	} else {
		log.Error().Err(err).Msg("Failed to get hostname")
	}

	// Get kernel version
	info.Kernel = runtime.GOOS + " " + runtime.GOARCH
	log.Debug().Str("default_kernel", info.Kernel).Msg("Default kernel info")

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("uname", "-r")
		output, err := cmd.Output()
		if err == nil {
			kernel := strings.TrimSpace(string(output))
			info.Kernel = "Linux " + kernel
			log.Debug().Str("kernel", info.Kernel).Msg("Got Linux kernel version")
		} else {
			log.Error().Err(err).Msg("Failed to run uname -r")
		}
	case "darwin":
		if uname, err := exec.Command("uname", "-r").Output(); err == nil {
			info.Kernel = "Darwin " + strings.TrimSpace(string(uname))
			log.Debug().Str("kernel", info.Kernel).Msg("Got Darwin kernel version")
		}
	}

	// Get uptime
	switch runtime.GOOS {
	case "linux":
		log.Debug().Msg("Getting Linux uptime from /proc/uptime")
		data, err := os.ReadFile("/proc/uptime")
		if err == nil {
			content := string(data)
			log.Debug().Str("proc_uptime_content", content).Msg("Read /proc/uptime")
			fields := strings.Fields(content)
			if len(fields) > 0 {
				uptime, parseErr := strconv.ParseFloat(fields[0], 64)
				if parseErr == nil {
					info.Uptime = int64(uptime)
					log.Debug().Int64("uptime_seconds", info.Uptime).Msg("Parsed uptime")
				} else {
					log.Error().Err(parseErr).Str("field", fields[0]).Msg("Failed to parse uptime")
				}
			} else {
				log.Error().Str("content", content).Msg("No fields in /proc/uptime")
			}
		} else {
			log.Error().Err(err).Msg("Failed to read /proc/uptime")
		}
	case "darwin":
		// macOS: use sysctl
		if output, err := exec.Command("sysctl", "-n", "kern.boottime").Output(); err == nil {
			// Parse: { sec = 1234567890, usec = 123456 }
			str := strings.TrimSpace(string(output))
			log.Debug().Str("sysctl_output", str).Msg("Got macOS boottime")
			if idx := strings.Index(str, "sec = "); idx != -1 {
				str = str[idx+6:]
				if idx := strings.Index(str, ","); idx != -1 {
					if sec, err := strconv.ParseInt(str[:idx], 10, 64); err == nil {
						info.Uptime = time.Now().Unix() - sec
						log.Debug().Int64("uptime_seconds", info.Uptime).Msg("Calculated macOS uptime")
					}
				}
			}
		}
	}

	// Get vnstat version
	cmd := exec.Command("vnstat", "--version")
	output, err := cmd.Output()
	if err == nil {
		versionOutput := string(output)
		log.Debug().Str("vnstat_version_output", versionOutput).Msg("vnstat version output")
		lines := strings.Split(versionOutput, "\n")
		if len(lines) > 0 {
			info.VnstatVersion = strings.TrimSpace(lines[0])
			log.Debug().Str("vnstat_version", info.VnstatVersion).Msg("Parsed vnstat version")
		}
	} else {
		log.Error().Err(err).Msg("Failed to get vnstat version")
	}

	// Get interface information
	interfaces, err := net.Interfaces()
	if err == nil {
		log.Debug().Msg("Getting network interfaces")
		for _, iface := range interfaces {
			// Skip loopback
			if iface.Flags&net.FlagLoopback != 0 {
				log.Debug().Str("interface", iface.Name).Msg("Skipping loopback interface")
				continue
			}

			log.Debug().Str("interface", iface.Name).Msg("Processing interface")

			ifaceInfo := InterfaceInfo{
				Name: iface.Name,
				IsUp: iface.Flags&net.FlagUp != 0,
			}

			// Get IP addresses
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				log.Debug().Str("interface", iface.Name).Int("addr_count", len(addrs)).Msg("Got addresses")
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
						ifaceInfo.IPAddress = ipnet.IP.String()
						log.Debug().Str("interface", iface.Name).Str("ip", ifaceInfo.IPAddress).Msg("Got IPv4 address")
						break
					}
				}
			} else if err != nil {
				log.Error().Err(err).Str("interface", iface.Name).Msg("Failed to get addresses")
			}

			// Try to get link speed (Linux)
			if runtime.GOOS == "linux" {
				// Check if this is a virtual/bridge interface
				bridgePath := fmt.Sprintf("/sys/class/net/%s/bridge", iface.Name)
				_, isBridge := os.Stat(bridgePath)
				
				// Check for common virtual interface patterns
				isVirtual := isBridge == nil || 
					strings.HasPrefix(iface.Name, "vmbr") || 
					strings.HasPrefix(iface.Name, "br") ||
					strings.HasPrefix(iface.Name, "virbr") ||
					strings.HasPrefix(iface.Name, "docker") ||
					strings.HasPrefix(iface.Name, "veth") ||
					strings.HasPrefix(iface.Name, "tap") ||
					strings.HasPrefix(iface.Name, "tun") ||
					strings.Contains(iface.Name, "bond")
				
				if isVirtual {
					// For virtual interfaces, set LinkSpeed to -1 to indicate it's virtual
					ifaceInfo.LinkSpeed = -1
					log.Debug().Str("interface", iface.Name).Msg("Detected virtual/bridge interface")
				} else {
					// For physical interfaces, read actual speed
					speedFile := fmt.Sprintf("/sys/class/net/%s/speed", iface.Name)
					data, err := os.ReadFile(speedFile)
					if err == nil {
						speedStr := strings.TrimSpace(string(data))
						log.Debug().Str("interface", iface.Name).Str("speed_raw", speedStr).Msg("Read link speed")
						if speed, err := strconv.Atoi(speedStr); err == nil && speed > 0 {
							ifaceInfo.LinkSpeed = speed
							log.Debug().Str("interface", iface.Name).Int("speed", speed).Msg("Got link speed")
						}
					} else {
						log.Debug().Err(err).Str("interface", iface.Name).Str("file", speedFile).Msg("No link speed available")
					}
				}
			}

			// Get vnstat interface alias if configured
			if output, err := exec.Command("vnstat", "--json", "-i", iface.Name).Output(); err == nil {
				var interfaceData map[string]interface{}
				if json.Unmarshal(output, &interfaceData) == nil {
					if interfaces, ok := interfaceData["interfaces"].([]interface{}); ok && len(interfaces) > 0 {
						if ifaceData, ok := interfaces[0].(map[string]interface{}); ok {
							if alias, ok := ifaceData["alias"].(string); ok {
								ifaceInfo.Alias = alias
							}
							if traffic, ok := ifaceData["traffic"].(map[string]interface{}); ok {
								if total, ok := traffic["total"].(map[string]interface{}); ok {
									if rx, ok := total["rx"].(float64); ok {
										if tx, ok := total["tx"].(float64); ok {
											ifaceInfo.BytesTotal = int64(rx + tx)
										}
									}
								}
							}
						}
					}
				}
			}

			info.Interfaces[iface.Name] = ifaceInfo
			log.Debug().
				Str("interface", iface.Name).
				Bool("is_up", ifaceInfo.IsUp).
				Str("ip", ifaceInfo.IPAddress).
				Int("speed", ifaceInfo.LinkSpeed).
				Msg("Added interface to info")
		}
	} else {
		log.Error().Err(err).Msg("Failed to get network interfaces")
	}

	// Log summary of collected info
	log.Info().
		Str("hostname", info.Hostname).
		Str("kernel", info.Kernel).
		Int64("uptime", info.Uptime).
		Str("vnstat_version", info.VnstatVersion).
		Int("interface_count", len(info.Interfaces)).
		Msg("System info collection complete")

	return info, nil
}

// handlePeakStats returns peak bandwidth statistics
func (a *Agent) handlePeakStats(c *gin.Context) {
	a.peakMu.RLock()
	stats := PeakStats{
		PeakRx:       a.peakRx,
		PeakTx:       a.peakTx,
		PeakRxString: formatBytesPerSecond(a.peakRx),
		PeakTxString: formatBytesPerSecond(a.peakTx),
		UpdatedAt:    time.Now(),
	}
	a.peakMu.RUnlock()

	c.JSON(http.StatusOK, stats)
}

// formatBytesPerSecond formats bytes per second to human readable string
func formatBytesPerSecond(bytes int) string {
	if bytes == 0 {
		return "0 B/s"
	}

	const k = 1024
	sizes := []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}

	i := 0
	bytesFloat := float64(bytes)
	for bytesFloat >= k && i < len(sizes)-1 {
		bytesFloat /= k
		i++
	}

	return fmt.Sprintf("%.2f %s", bytesFloat, sizes[i])
}

// handleHardwareStats returns hardware statistics
func (a *Agent) handleHardwareStats(c *gin.Context) {
	stats, err := a.getHardwareStats()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get hardware stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to get hardware stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// getHardwareStats gathers hardware statistics
func (a *Agent) getHardwareStats() (*HardwareStats, error) {
	log.Debug().
		Strs("disk_includes", a.config.DiskIncludes).
		Strs("disk_excludes", a.config.DiskExcludes).
		Msg("Starting getHardwareStats with disk filters")

	stats := &HardwareStats{
		UpdatedAt: time.Now(),
	}

	// Get CPU info for model name and frequency
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		// Use the first CPU entry for model name and frequency
		stats.CPU.Model = cpuInfo[0].ModelName
		stats.CPU.Frequency = cpuInfo[0].Mhz
		
		log.Debug().
			Str("model", stats.CPU.Model).
			Float64("freq", stats.CPU.Frequency).
			Int("cpu_entries", len(cpuInfo)).
			Msg("Got CPU info")
	} else {
		log.Error().Err(err).Msg("Failed to get CPU info")
	}

	// Get physical CPU cores using cpu.Counts(false)
	physicalCores, err := cpu.Counts(false)
	if err == nil {
		stats.CPU.Cores = physicalCores
		log.Debug().Int("physical_cores", physicalCores).Msg("Got physical CPU core count")
	} else {
		log.Error().Err(err).Msg("Failed to get physical CPU count")
		// Fallback: try to get from cpuInfo if available
		if len(cpuInfo) > 0 && cpuInfo[0].Cores > 0 {
			stats.CPU.Cores = int(cpuInfo[0].Cores)
		}
	}

	// Get logical CPU threads using cpu.Counts(true)
	logicalThreads, err := cpu.Counts(true)
	if err == nil {
		stats.CPU.Threads = logicalThreads
		log.Debug().Int("logical_threads", logicalThreads).Msg("Got logical CPU thread count")
	} else {
		log.Error().Err(err).Msg("Failed to get logical CPU count")
	}
	
	// Handle edge case for containers (like LXC) where threads might be less than cores
	// In containers, the logical count may reflect container limits
	if stats.CPU.Threads < stats.CPU.Cores && stats.CPU.Threads > 0 {
		log.Debug().
			Int("original_cores", stats.CPU.Cores).
			Int("threads", stats.CPU.Threads).
			Msg("Threads less than cores (possibly in container), adjusting cores to match threads")
		stats.CPU.Cores = stats.CPU.Threads
	}

	// Get CPU usage percentage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		stats.CPU.UsagePercent = cpuPercent[0]
		log.Debug().Float64("usage", cpuPercent[0]).Msg("Got CPU usage")
	} else {
		log.Error().Err(err).Msg("Failed to get CPU usage")
	}

	// Get load average (Unix-like systems only)
	if runtime.GOOS != "windows" {
		loadAvg, err := getLoadAverage()
		if err == nil {
			stats.CPU.LoadAvg = loadAvg
			log.Debug().Floats64("load_avg", loadAvg).Msg("Got load average")
		} else {
			log.Error().Err(err).Msg("Failed to get load average")
		}
	}

	// Get memory stats
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory.Total = vmStat.Total
		stats.Memory.Free = vmStat.Free
		stats.Memory.Available = vmStat.Available
		stats.Memory.Cached = vmStat.Cached
		stats.Memory.Buffers = vmStat.Buffers
		
		// Get ZFS ARC size if available
		zfsArcSize := getZFSARCSize()
		stats.Memory.ZFSArc = zfsArcSize
		
		// Calculate used memory
		// On Linux, the Used field from gopsutil includes cache/buffers
		// We need to be careful about how we calculate "used" memory
		if runtime.GOOS == "linux" {
			// Linux calculation: Total - Free - Buffers - Cached
			// This gives us the actual application memory usage
			stats.Memory.Used = vmStat.Total - vmStat.Free
			// The UsedPercent should reflect total used including cache/buffers
			stats.Memory.UsedPercent = float64(stats.Memory.Used) / float64(vmStat.Total) * 100
		} else {
			// For other systems, use the provided values
			stats.Memory.Used = vmStat.Used
			stats.Memory.UsedPercent = vmStat.UsedPercent
		}
		
		log.Debug().
			Uint64("total", vmStat.Total).
			Uint64("used", stats.Memory.Used).
			Uint64("free", vmStat.Free).
			Uint64("available", vmStat.Available).
			Uint64("cached", vmStat.Cached).
			Uint64("buffers", vmStat.Buffers).
			Uint64("zfs_arc", zfsArcSize).
			Float64("percent", stats.Memory.UsedPercent).
			Msg("Got memory stats")
	} else {
		log.Error().Err(err).Msg("Failed to get memory stats")
	}

	// Get swap memory stats
	swapStat, err := mem.SwapMemory()
	if err == nil {
		stats.Memory.SwapTotal = swapStat.Total
		stats.Memory.SwapUsed = swapStat.Used
		stats.Memory.SwapPercent = swapStat.UsedPercent
		log.Debug().
			Uint64("total", swapStat.Total).
			Uint64("used", swapStat.Used).
			Float64("percent", swapStat.UsedPercent).
			Msg("Got swap stats")
	} else {
		log.Debug().Err(err).Msg("Failed to get swap stats (may be normal if no swap)")
	}

	// Get disk stats (include all filesystems, including fuse/virtual ones)
	partitions, err := disk.Partitions(true)
	if err == nil {
		log.Debug().Int("partition_count", len(partitions)).Msg("Got disk partitions")
		
		// Build a map of device names to SMART info
		deviceInfoMap := make(map[string]struct{ model, serial string })
		devicePaths := a.getDevicePaths()
		for _, devicePath := range devicePaths {
			model, serial := a.getDiskInfo(devicePath)
			if model != "" || serial != "" {
				// Store info for both the raw device and partition names
				baseName := filepath.Base(devicePath)
				deviceInfoMap[devicePath] = struct{ model, serial string }{model, serial}
				deviceInfoMap["/dev/"+baseName] = struct{ model, serial string }{model, serial}
				// Also store without partition suffix for matching
				if strings.Contains(baseName, "disk") {
					// macOS style: /dev/diskX -> store for matching /dev/diskXsY
					deviceInfoMap["/dev/"+baseName] = struct{ model, serial string }{model, serial}
				}
			}
		}
		
		for _, partition := range partitions {
			// Check if this disk should be included based on filters
			if !a.shouldIncludeDisk(partition.Mountpoint, partition.Device, partition.Fstype) {
				log.Debug().
					Str("mount", partition.Mountpoint).
					Str("device", partition.Device).
					Str("fstype", partition.Fstype).
					Msg("Skipping disk based on filter rules")
				continue
			}

			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				log.Debug().Err(err).Str("mount", partition.Mountpoint).Msg("Failed to get disk usage")
				continue
			}

			// Skip if disk is too small (less than 1GB)
			if usage.Total < 1024*1024*1024 {
				log.Debug().
					Str("mount", partition.Mountpoint).
					Uint64("total", usage.Total).
					Msg("Skipping small disk")
				continue
			}
			
			diskStat := DiskStats{
				Path:        partition.Mountpoint,
				Device:      partition.Device,
				Fstype:      partition.Fstype,
				Total:       usage.Total,
				Used:        usage.Used,
				Free:        usage.Free,
				UsedPercent: usage.UsedPercent,
			}
			
			// Try to find SMART info for this device
			// First try exact match
			if info, ok := deviceInfoMap[partition.Device]; ok {
				diskStat.Model = info.model
				diskStat.Serial = info.serial
			} else {
				// Try to match by base device name (remove partition number)
				baseDevice := partition.Device
				// Remove partition suffix (e.g., /dev/sda1 -> /dev/sda)
				if idx := strings.LastIndexAny(baseDevice, "0123456789"); idx > 0 && idx == len(baseDevice)-1 {
					// Remove trailing numbers
					for idx > 0 && baseDevice[idx-1] >= '0' && baseDevice[idx-1] <= '9' {
						idx--
					}
					baseDevice = baseDevice[:idx]
				}
				// Also handle p1, p2 style partitions (e.g., /dev/nvme0n1p1 -> /dev/nvme0n1)
				if strings.Contains(baseDevice, "p") && len(baseDevice) > 2 {
					if idx := strings.LastIndex(baseDevice, "p"); idx > 0 {
						if idx < len(baseDevice)-1 && baseDevice[idx+1] >= '0' && baseDevice[idx+1] <= '9' {
							baseDevice = baseDevice[:idx]
						}
					}
				}
				
				if info, ok := deviceInfoMap[baseDevice]; ok {
					diskStat.Model = info.model
					diskStat.Serial = info.serial
				}
			}

			stats.Disks = append(stats.Disks, diskStat)

			log.Debug().
				Str("path", partition.Mountpoint).
				Str("device", partition.Device).
				Str("model", diskStat.Model).
				Uint64("total", usage.Total).
				Float64("percent", usage.UsedPercent).
				Msg("Added disk to stats")
		}
		log.Debug().Int("disk_count", len(stats.Disks)).Msg("Finished processing disks")
	} else {
		log.Error().Err(err).Msg("Failed to get disk partitions")
	}

	// Get temperature sensors
	temps, err := sensors.TemperaturesWithContext(context.Background())
	if err == nil {
		log.Debug().Int("sensor_count", len(temps)).Msg("Got temperature sensors")
		for _, temp := range temps {
			// Skip sensors with zero or invalid readings
			if temp.Temperature <= 0 || temp.Temperature > 200 {
				log.Trace().
					Str("sensor", temp.SensorKey).
					Float64("temp", temp.Temperature).
					Msg("Skipping invalid temperature reading")
				continue
			}

			// Filter out redundant PMU sensors - only keep the most important ones
			sensorKey := temp.SensorKey
			if strings.Contains(sensorKey, "PMU") {
				// For PMU sensors, only keep a representative sample
				if strings.Contains(sensorKey, "tdev") && !strings.HasSuffix(sensorKey, "tdev1") {
					// Skip most tdev sensors, only keep tdev1 from each PMU
					continue
				}
				if strings.Contains(sensorKey, "tdie") && !strings.HasSuffix(sensorKey, "tdie1") {
					// Skip most tdie sensors, only keep tdie1 from each PMU
					continue
				}
			}

			stats.Temperature = append(stats.Temperature, TemperatureStats{
				SensorKey:   temp.SensorKey,
				Temperature: temp.Temperature,
				Label:       "", // gopsutil doesn't provide labels
				Critical:    temp.High,
			})

			log.Debug().
				Str("sensor", temp.SensorKey).
				Float64("temp", temp.Temperature).
				Float64("critical", temp.High).
				Msg("Added temperature sensor")
		}
		log.Debug().Int("temp_sensor_count", len(stats.Temperature)).Msg("Finished processing temperature sensors")
	} else {
		log.Debug().Err(err).Msg("Failed to get temperature sensors (may be normal on some systems)")
	}

	// Get disk temperatures via SMART (Linux: SATA+NVMe, macOS: NVMe only, requires privileges)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		hddTemps := a.getHDDTemperatures()
		if len(hddTemps) > 0 {
			stats.Temperature = append(stats.Temperature, hddTemps...)
			log.Debug().Int("hdd_temp_count", len(hddTemps)).Msg("Added disk temperature sensors")
		}
	}

	// Log final summary
	log.Info().
		Float64("cpu_usage", stats.CPU.UsagePercent).
		Int("cpu_threads", stats.CPU.Threads).
		Float64("mem_percent", stats.Memory.UsedPercent).
		Int("disk_count", len(stats.Disks)).
		Int("temp_count", len(stats.Temperature)).
		Msg("Hardware stats collection complete")

	return stats, nil
}

// getZFSARCSize gets the ZFS ARC size on systems using ZFS
func getZFSARCSize() uint64 {
	// Check for ZFS ARC stats file (Linux)
	arcStatsPath := "/proc/spl/kstat/zfs/arcstats"
	data, err := os.ReadFile(arcStatsPath)
	if err != nil {
		// Not a ZFS system or no permissions
		return 0
	}

	// Parse the arcstats file to find the size
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "size" {
			size, err := strconv.ParseUint(fields[2], 10, 64)
			if err == nil {
				return size
			}
		}
	}

	return 0
}

// getLoadAverage gets system load averages (Unix-like systems)
func getLoadAverage() ([]float64, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("load average not available on Windows")
	}

	loadAvg, err := load.Avg()
	if err != nil {
		return nil, err
	}

	return []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}, nil
}

// matchPath checks if a path matches a pattern, supporting both glob and prefix patterns
func matchPath(pattern, path string) bool {
	// Check if pattern ends with * for prefix matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	
	// Check if pattern contains any glob characters
	if strings.ContainsAny(pattern, "*?[]") {
		// Use filepath.Match for single-component patterns
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also try matching just the basename
		matched, err = filepath.Match(pattern, filepath.Base(path))
		return err == nil && matched
	}
	
	// Otherwise, exact match
	return pattern == path
}

// shouldIncludeDisk determines if a disk should be included in monitoring based on filter rules
func (a *Agent) shouldIncludeDisk(mountpoint, device, fstype string) bool {
	// First check if it's explicitly included - this takes precedence
	for _, pattern := range a.config.DiskIncludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk included by pattern match")
			return true
		}
	}

	// Then check if it's explicitly excluded
	for _, pattern := range a.config.DiskExcludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk excluded by pattern match")
			return false
		}
	}

	// Skip special filesystems by default (unless explicitly included above)
	if strings.HasPrefix(mountpoint, "/snap") ||
		strings.HasPrefix(mountpoint, "/run") ||
		strings.HasPrefix(mountpoint, "/dev") ||
		strings.HasPrefix(mountpoint, "/proc") ||
		strings.HasPrefix(mountpoint, "/sys") ||
		strings.HasPrefix(mountpoint, "/var/lib/docker/overlay") ||
		strings.HasPrefix(mountpoint, "/var/lib/containers/storage/overlay") ||
		strings.HasPrefix(device, "tmpfs") ||
		strings.HasPrefix(device, "devfs") ||
		strings.HasPrefix(device, "udev") ||
		strings.HasPrefix(device, "overlay") ||
		fstype == "squashfs" ||
		fstype == "devtmpfs" ||
		fstype == "overlay" ||
		fstype == "proc" ||
		fstype == "sysfs" ||
		fstype == "cgroup" ||
		fstype == "cgroup2" {
		return false
	}

	// Include everything else by default
	return true
}


// getDevicePaths returns a list of device paths to check for SMART data
func (a *Agent) getDevicePaths() []string {
	var devicePaths []string

	if runtime.GOOS == "darwin" {
		// macOS: Look for disk devices in /dev
		entries, err := os.ReadDir("/dev")
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read /dev directory")
			return devicePaths
		}

		for _, entry := range entries {
			name := entry.Name()
			// macOS uses disk0, disk1, etc. and rdisk0, rdisk1, etc.
			// We want the raw disk devices (rdisk) for SMART access
			if strings.HasPrefix(name, "rdisk") && !strings.Contains(name, "s") {
				// Skip partitions (rdisk0s1, rdisk1s2, etc.)
				devicePaths = append(devicePaths, filepath.Join("/dev", name))
			}
			// Also check for nvme devices that might exist
			if strings.HasPrefix(name, "nvme") && !strings.Contains(name, "p") {
				devicePaths = append(devicePaths, filepath.Join("/dev", name))
			}
		}
	} else {
		// Linux: Look for traditional device names
		entries, err := os.ReadDir("/dev")
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read /dev directory")
			return devicePaths
		}

		for _, entry := range entries {
			name := entry.Name()
			// Look for disk devices (sda, sdb, nvme0n1, etc.)
			if !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "nvme") && !strings.HasPrefix(name, "hd") {
				continue
			}
			
			// Skip partitions (sda1, sdb2, nvme0n1p1, etc.)
			if strings.Contains(name, "p") && len(name) > 4 {
				continue
			}
			if len(name) > 3 && name[len(name)-1] >= '0' && name[len(name)-1] <= '9' {
				continue
			}

			devicePaths = append(devicePaths, filepath.Join("/dev", name))
		}
	}

	return devicePaths
}
