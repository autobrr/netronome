// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	// Enable console writer with colors
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// Set global logger
	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set log level based on environment
	if os.Getenv("GIN_MODE") == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Get returns the global logger instance
func Get() zerolog.Logger {
	return log.Logger
}
