// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
		FormatLevel: func(i interface{}) string {
			if ll, ok := i.(string); ok {
				switch ll {
				case "trace":
					return "\033[36m" + strings.ToUpper(ll) + "\033[0m" // Cyan
				case "debug":
					return "\033[33m" + strings.ToUpper(ll) + "\033[0m" // Orange
				case "info":
					return "\033[34m" + strings.ToUpper(ll) + "\033[0m" // Blue
				case "warn":
					return "\033[33m" + strings.ToUpper(ll) + "\033[0m" // Yellow
				case "error":
					return "\033[31m" + strings.ToUpper(ll) + "\033[0m" // Red
				default:
					return strings.ToUpper(ll)
				}
			}
			return "???"
		},
	}

	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msgf("Logger initialized in debug mode (GIN_MODE=%s)", ginMode)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Info().Msg("Logger initialized in production mode")
	}
}

func Get() zerolog.Logger {
	return log.Logger
}
