// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package broadcaster

import "github.com/autobrr/netronome/internal/types"

type Broadcaster interface {
	BroadcastUpdate(types.SpeedUpdate)
}
