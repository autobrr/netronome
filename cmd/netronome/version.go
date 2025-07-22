// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"

	appversion "github.com/autobrr/netronome/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("netronome version: %s\n", appversion.Version)
		if appversion.Commit != "unknown" {
			fmt.Printf("Git commit:        %s\n", appversion.Commit)
		}
		if appversion.BuildTime != "unknown" {
			fmt.Printf("Build time:        %s\n", appversion.BuildTime)
		}
		fmt.Printf("Go version:        %s\n", runtime.Version())
		fmt.Printf("OS/Arch:           %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
	DisableFlagsInUseLine: true,
}

func SetVersion(v, bt, c string) {
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			v = info.Main.Version
		}
	}
	appversion.Set(v, bt, c)
}

func init() {
	versionCmd.SetUsageTemplate(`Usage:
  {{.CommandPath}}

Prints the version and build time information for netronome.
`)
}
