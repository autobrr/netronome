// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import "time"

type SpeedTestConfig struct {
	Timeout time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type PaginationConfig struct {
	DefaultTimeRange string
	DefaultPage      int
	DefaultLimit     int
}

type Config struct {
	SpeedTest  SpeedTestConfig
	CORS       CORSConfig
	Pagination PaginationConfig
}

func NewConfig() *Config {
	return &Config{
		SpeedTest: SpeedTestConfig{
			Timeout: 30 * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{
				"Content-Type",
				"Content-Length",
				"Accept-Encoding",
				"X-CSRF-Token",
				"Authorization",
				"accept",
				"origin",
				"Cache-Control",
				"X-Requested-With",
			},
		},
		Pagination: PaginationConfig{
			DefaultTimeRange: "1w",
			DefaultPage:      1,
			DefaultLimit:     100,
		},
	}
}
