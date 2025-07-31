// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
)

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
