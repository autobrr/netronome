// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build !nosmart && (linux || darwin)
// +build !nosmart
// +build linux darwin

package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/rs/zerolog/log"
)

// swapBytes swaps pairs of bytes in ATA strings which are stored in a special format
func swapBytes(b []byte) []byte {
	swapped := make([]byte, len(b))
	for i := 0; i < len(b)-1; i += 2 {
		swapped[i] = b[i+1]
		swapped[i+1] = b[i]
	}
	if len(b)%2 == 1 {
		swapped[len(b)-1] = b[len(b)-1]
	}
	return swapped
}

// getDiskInfo retrieves disk model/serial info for a device path
func (a *Agent) getDiskInfo(devicePath string) (model string, serial string) {
	dev, err := smart.Open(devicePath)
	if err != nil {
		return "", ""
	}
	defer dev.Close()

	// Try different device types to get identify information
	switch d := dev.(type) {
	case *smart.SataDevice:
		if ident, err := d.Identify(); err == nil {
			// ATA strings have byte pairs swapped
			model = strings.TrimSpace(string(swapBytes(ident.ModelNumberRaw[:])))
			serial = strings.TrimSpace(string(swapBytes(ident.SerialNumberRaw[:])))
		}
	case *smart.NVMeDevice:
		if ctrl, _, err := d.Identify(); err == nil {
			// NVMe strings are stored normally
			model = strings.TrimSpace(string(ctrl.ModelNumberRaw[:]))
			serial = strings.TrimSpace(string(ctrl.SerialNumberRaw[:]))
		}
	}
	
	return model, serial
}

// getHDDTemperatures retrieves disk temperatures via SMART data
func (a *Agent) getHDDTemperatures() []TemperatureStats {
	var temps []TemperatureStats

	// Get device paths based on platform
	devicePaths := a.getDevicePaths()
	
	for _, devicePath := range devicePaths {
		name := filepath.Base(devicePath)
		
		// Try to open the device with SMART
		dev, err := smart.Open(devicePath)
		if err != nil {
			log.Trace().Str("device", devicePath).Err(err).Msg("Failed to open device for SMART (may need root or unsupported on this platform)")
			continue
		}
		
		// Get temperature using the generic attributes API
		attrs, err := dev.ReadGenericAttributes()
		if err != nil {
			dev.Close()
			log.Trace().Str("device", devicePath).Err(err).Msg("Failed to read generic attributes")
			continue
		}
		
		// Check if we got a valid temperature
		if attrs != nil && attrs.Temperature > 0 && attrs.Temperature < 100 {
			deviceType := "HDD"
			modelName := ""
			
			// Try to get model information
			switch d := dev.(type) {
			case *smart.SataDevice:
				if ident, err := d.Identify(); err == nil {
					// ATA strings have byte pairs swapped
					modelName = strings.TrimSpace(string(swapBytes(ident.ModelNumberRaw[:])))
					// Check rotation rate to determine if SSD (0 RPM) or HDD
					// Note: This is a simplified check, some SSDs may not report 0
					deviceType = "HDD" // Default to HDD for SATA
				}
			case *smart.NVMeDevice:
				deviceType = "NVMe"
				if ctrl, _, err := d.Identify(); err == nil {
					// NVMe strings are stored normally
					modelName = strings.TrimSpace(string(ctrl.ModelNumberRaw[:]))
				}
			}
			
			label := fmt.Sprintf("%s %s", deviceType, strings.ToUpper(name))
			if modelName != "" {
				// Include both model name and device identifier
				label = fmt.Sprintf("%s (%s)", modelName, strings.ToUpper(name))
			}
			
			temps = append(temps, TemperatureStats{
				SensorKey:   fmt.Sprintf("smart_%s", name),
				Temperature: float64(attrs.Temperature),
				Label:       label,
				Critical:    60.0, // HDDs typically warn at 60°C, SSDs at 70°C
			})

			log.Debug().
				Str("device", name).
				Uint64("temp", attrs.Temperature).
				Str("type", deviceType).
				Str("model", modelName).
				Msg("Added disk temperature from SMART")
		}
		
		dev.Close()
	}

	return temps
}