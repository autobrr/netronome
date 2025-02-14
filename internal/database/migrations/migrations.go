// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"embed"
	"fmt"
	"io/fs"
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

	log.Debug().
		Str("basePath", basePath).
		Str("suffix", suffix).
		Msg("Looking for migration files")

	entries, err := SchemaMigrations.ReadDir(basePath)
	if err != nil {
		log.Error().Err(err).Str("basePath", basePath).Msg("Failed to read migrations directory")
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	log.Debug().Int("entryCount", len(entries)).Msg("Found entries in migrations directory")

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
	n := len(files)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if getMigrationVersion(files[j]) > getMigrationVersion(files[j+1]) {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}
}

// getMigrationVersion extracts the version number from a migration filename
func getMigrationVersion(fileName string) int {
	parts := strings.Split(fileName, "_")
	if len(parts) > 0 {
		version := strings.TrimPrefix(parts[0], "0")
		if v, err := parseInt(version); err == nil {
			return v
		}
	}
	return 0
}

func parseInt(s string) (int, error) {
	var result int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(ch-'0')
	}
	return result, nil
}

func ReadMigration(fileName string) ([]byte, error) {
	content, err := fs.ReadFile(SchemaMigrations, fileName)
	if err != nil {
		log.Error().Err(err).Str("file", fileName).Msg("Failed to read migration file")
		return nil, err
	}

	log.Debug().
		Str("file", fileName).
		Int("contentLength", len(content)).
		Msg("Successfully read migration content")

	return content, nil
}
