// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"cmp"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

//go:embed postgres/*.sql sqlite/*.sql
var SchemaMigrations embed.FS

type DatabaseType string

const (
	SQLite   DatabaseType = "sqlite"
	Postgres DatabaseType = "postgres"
)

// GetMigrationFiles returns the appropriate migration files for the given database type
func GetMigrationFiles(dbType DatabaseType) ([]string, error) {
	var basePath string
	var suffix string
	switch dbType {
	case Postgres:
		basePath = "postgres"
		suffix = "_postgres.sql"
	case SQLite:
		basePath = "sqlite"
		suffix = ".sql"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Only log debug info if not in test environment
	isTest := strings.Contains(os.Args[0], ".test") || strings.HasSuffix(os.Args[0], "/test")
	if !isTest {
		log.Debug().
			Str("basePath", basePath).
			Str("suffix", suffix).
			Msg("Looking for migration files")
	}

	entries, err := SchemaMigrations.ReadDir(basePath)
	if err != nil {
		log.Error().Err(err).Str("basePath", basePath).Msg("Failed to read migrations directory")
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	if !isTest {
		log.Debug().Int("entryCount", len(entries)).Msg("Found entries in migrations directory")
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			filePath := fmt.Sprintf("%s/%s", basePath, entry.Name())

			_, err := fs.ReadFile(SchemaMigrations, filePath)
			if err != nil {
				log.Error().Err(err).Str("file", filePath).Msg("Failed to read migration file")
			}

			files = append(files, filePath)
		}
	}

	// log.Trace().
	//	Strs("files", files).
	//	Int("fileCount", len(files)).
	//	Msg("Final migration files list")

	sortMigrationFiles(files)

	return files, nil
}

// sortMigrationFiles sorts migration files by their version number
func sortMigrationFiles(files []string) {
	slices.SortStableFunc(files, func(a, b string) int {
		return cmp.Compare(getMigrationVersion(a), getMigrationVersion(b))
	})
}

// getMigrationVersion extracts the version number from a migration file path
func getMigrationVersion(filePath string) int {
	prefix, _, _ := strings.Cut(path.Base(filePath), "_")
	v, err := strconv.Atoi(prefix)
	if err != nil {
		return 0
	}
	return v
}
