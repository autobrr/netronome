package migrations

import "embed"

var (
	//go:embed *.sql
	SchemaMigrations embed.FS
)

// MigrationFiles holds all migration file names in order
var MigrationFiles = []string{
	"001_initial_schema.sql",
	"002_add_jitter.sql",
}
