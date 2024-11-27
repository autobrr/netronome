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
	"003_add_test_type.sql",
	"004_add_is_scheduled_column.sql",
	"005_drop_user_id_and_cv.sql",
}
