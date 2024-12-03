// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed postgres/*.sql sqlite/*.sql
var SchemaMigrations embed.FS

// DatabaseType represents the type of database being used
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

	entries, err := SchemaMigrations.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			files = append(files, fmt.Sprintf("%s/%s", basePath, entry.Name()))
		}
	}

	// Sort files by version number to ensure correct order
	sortMigrationFiles(files)

	return files, nil
}

// sortMigrationFiles sorts migration files by their version number
func sortMigrationFiles(files []string) {
	// Simple bubble sort since we have a small number of files
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

// parseInt safely converts a string to an integer
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

// ReadMigration reads the content of a migration file
func ReadMigration(fileName string) ([]byte, error) {
	return fs.ReadFile(SchemaMigrations, fileName)
}
