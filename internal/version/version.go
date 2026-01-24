// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package version

var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func Set(v, bt, c string) {
	Version = v
	BuildTime = bt
	Commit = c
}
