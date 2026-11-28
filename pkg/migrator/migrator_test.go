package migrator

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFileMigrationExecFailureRollsBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*Migrator, *Migration) error
	}{
		{
			name: "initial",
			run:  (*Migrator).migrateInitialSchema,
		},
		{
			name: "subsequent",
			run: func(m *Migrator, migration *Migration) error {
				return m.migrate(1, migration)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			file := filepath.Join(t.TempDir(), "migration.sql")
			if err := os.WriteFile(file, []byte("CREATE TABLE first; INVALID SQL;"), 0o600); err != nil {
				t.Fatal(err)
			}

			mock.ExpectBegin()
			mock.ExpectExec("CREATE TABLE first; INVALID SQL;").WillReturnError(errors.New("statement failed"))
			mock.ExpectRollback()

			m := NewMigrate(db)
			if err := tc.run(m, &Migration{Name: "test", File: file}); err == nil {
				t.Fatal("expected migration error")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommitFailureIsReturned(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*Migrator, *Migration) error
	}{
		{
			name: "initial",
			run:  (*Migrator).migrateInitialSchema,
		},
		{
			name: "subsequent",
			run: func(m *Migrator, migration *Migration) error {
				return m.migrate(1, migration)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			commitErr := errors.New("commit failed")
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO schema_migrations").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit().WillReturnError(commitErr)

			m := NewMigrate(db)
			migration := &Migration{Name: "test", RunTx: func(*sql.Tx) error { return nil }}
			err = tc.run(m, migration)
			if err == nil || !strings.Contains(err.Error(), commitErr.Error()) {
				t.Fatalf("expected commit error, got %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
