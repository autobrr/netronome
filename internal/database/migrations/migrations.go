// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import "embed"

var (
	//go:embed *.sql
	SchemaMigrations embed.FS
)

// MigrationFiles holds all migration file names in order
var MigrationFiles = []string{
	"001_initial_schema.sql",
	"002_auth_schema.sql",
}
