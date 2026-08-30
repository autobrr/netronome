// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package license

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

const checkInterval = 24 * time.Hour

// Checker revalidates the stored license on a ticker.
//
// ponytail: no in-memory validity/grace state. Validity is derived from the
// stored blob (Status + LastValidated + offlineGracePeriod), so it survives
// restarts and there is nothing to keep in sync.
type Checker struct {
	service  *Service
	interval time.Duration
}

func NewChecker(service *Service) *Checker {
	return &Checker{service: service, interval: checkInterval}
}

// StartPeriodicChecks blocks until ctx is done. Run it in a goroutine.
func (c *Checker) StartPeriodicChecks(ctx context.Context) {
	c.check(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.check(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Checker) check(ctx context.Context) {
	lic, err := c.service.Validate(ctx)
	if err != nil {
		// Transient failure. Validate left the stored license alone, so
		// entitlement rides the offline grace period.
		log.Debug().Err(err).Msg("license revalidation failed")
		return
	}

	if lic == nil {
		return
	}

	log.Debug().
		Str("status", lic.Status).
		Bool("entitled", lic.Entitled(time.Now())).
		Msg("license revalidated")
}
