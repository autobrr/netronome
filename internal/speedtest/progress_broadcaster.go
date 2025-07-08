// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"github.com/autobrr/netronome/internal/broadcaster"
	"github.com/autobrr/netronome/internal/types"
)

type DefaultProgressBroadcaster struct {
	broadcaster broadcaster.Broadcaster
}

func NewProgressBroadcaster(broadcaster broadcaster.Broadcaster) *DefaultProgressBroadcaster {
	return &DefaultProgressBroadcaster{
		broadcaster: broadcaster,
	}
}

func (p *DefaultProgressBroadcaster) BroadcastUpdate(update types.SpeedUpdate) {
	if p.broadcaster != nil {
		p.broadcaster.BroadcastUpdate(update)
	}
}

