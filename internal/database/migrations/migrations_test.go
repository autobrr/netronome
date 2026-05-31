// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"path"
	"strconv"
	"strings"
	"testing"
)

// TestNoDuplicateMigrationVersions guards against two migration files sharing a
// numeric version prefix. The migrator (pkg/migrator) applies migrations by
// position relative to the count already applied, so a duplicate version makes
// the sorted order ambiguous and silently skips a migration on upgrade (the new
// file gets ordered before an already-applied one and is never run).
//
// The version is parsed from the file basename here rather than via the
// production getMigrationVersion, which does not strip the directory prefix.
func TestNoDuplicateMigrationVersions(t *testing.T) {
	for _, dbType := range []DatabaseType{SQLite, Postgres} {
		files, err := GetMigrationFiles(dbType)
		if err != nil {
			t.Fatalf("%s: failed to get migration files: %v", dbType, err)
		}

		seen := make(map[int]string, len(files))
		for _, file := range files {
			base := path.Base(file)
			prefix, _, _ := strings.Cut(base, "_")
			version, err := strconv.Atoi(prefix)
			if err != nil {
				t.Errorf("%s: migration %q has no numeric version prefix", dbType, file)
				continue
			}
			if prev, ok := seen[version]; ok {
				t.Errorf("%s: duplicate migration version %d: %q and %q", dbType, version, prev, file)
			}
			seen[version] = file
		}
	}
}
