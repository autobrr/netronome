// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build nosmart || (!linux && !darwin)
// +build nosmart !linux,!darwin

package agent

// getDiskInfo stub for platforms without SMART support
func (a *Agent) getDiskInfo(_ string) (model string, serial string) {
	return "", ""
}

// getHDDTemperatures stub for platforms without SMART support
func (a *Agent) getHDDTemperatures() []TemperatureStats {
	return []TemperatureStats{}
}