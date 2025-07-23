// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

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
