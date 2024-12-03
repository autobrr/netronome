// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrator

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
)

const DefaultTableName = "schema_migrations"

type Migrator struct {
	db        *sql.DB
	tableName string
	logger    Logger
	embedFS   *embed.FS

	initialSchemaFile string
	initialSchema     string

	migrations      []*Migration
	migrationLookup map[int]*Migration
}

type Option func(migrate *Migrator)

func WithTableName(table string) Option {
	return func(migrate *Migrator) {
		migrate.tableName = table
	}
}

func WithSchemaString(schema string) Option {
	return func(migrate *Migrator) {
		migrate.initialSchema = schema
	}
}

func WithSchemaFile(file string) Option {
	return func(migrate *Migrator) {
		migrate.initialSchemaFile = file
	}
}

func WithEmbedFS(embedFS embed.FS) Option {
	return func(migrate *Migrator) {
		migrate.embedFS = &embedFS
	}
}

// Logger interface
type Logger interface {
	Printf(string, ...interface{})
}

// LoggerFunc adapts Logger and any third party logger
type LoggerFunc func(string, ...interface{})

// Printf implements Logger interface
func (f LoggerFunc) Printf(msg string, args ...interface{}) {
	f(msg, args...)
}

func WithLogger(logger Logger) Option {
	return func(migrate *Migrator) {
		migrate.logger = logger
	}
}

func NewMigrate(db *sql.DB, opts ...Option) *Migrator {
	m := &Migrator{
		db:                db,
		tableName:         DefaultTableName,
		logger:            log.New(io.Discard, "migrator: ", 0),
		initialSchema:     "",
		initialSchemaFile: "",
		migrations:        make([]*Migration, 0),
		migrationLookup:   map[int]*Migration{},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

type Migration struct {
	id int

	Name  string
	File  string
	Run   func(db *sql.DB) error
	RunTx func(db *sql.Tx) error

	db *sql.DB
}

func (m *Migration) String() string {
	return m.Name
}

func (m *Migration) Id() int {
	return m.id
}

func (m *Migrator) TableDrop(table string) error {
	if _, err := m.db.Exec(fmt.Sprintf(`DROP TABLE "%s"`, table)); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) Add(mi ...*Migration) {
	for _, migration := range mi {
		migration.db = m.db

		m.migrations = append(m.migrations, migration)
		m.migrationLookup[migration.id] = migration
	}
}

func (m *Migrator) Exec(query string, args ...string) error {
	if _, err := m.db.Exec(query, args); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) BeginTx() (*sql.Tx, error) {
	return m.db.BeginTx(context.Background(), nil)
}

func (m *Migrator) CountApplied() (int, error) {
	row := m.db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", m.tableName))
	if row.Err() != nil {
		return 0, row.Err()
	}

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (m *Migrator) Pending() ([]*Migration, error) {
	count, err := m.CountApplied()
	if err != nil {
		return nil, err
	}

	return m.migrations[count:len(m.migrations)], nil
}

func (m *Migrator) Migrate() error {
	migrationsTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    id       INT8 NOT NULL,
	version  VARCHAR(255) NOT NULL,
    datetime VARCHAR(255) NOT NULL,
	PRIMARY KEY(id)
);`, m.tableName)

	_, err := m.db.Exec(migrationsTable)
	if err != nil {
		return errors.Wrapf(err, "migrator: could not apply version table: %q", m.tableName)
	}

	appliedCount, err := m.CountApplied()
	if err != nil {
		return errors.Wrap(err, "migrator: could not get applied migrations count")
	}

	if appliedCount == 0 {
		m.logger.Printf("preparing to apply base schema migration")

		if len(m.migrations) == 0 {
			return errors.New("migrator: no migrations defined")
		}

		err = m.migrateInitialSchema(m.migrations[0])
		if err != nil {
			return errors.Wrap(err, "migrator: could not apply base schema")
		}

		m.logger.Printf("successfully applied all migrations!")

		return nil
	}

	// TODO check base schema migrations++
	//if appliedCount-1 > len(m.migrations) {
	if appliedCount > len(m.migrations) {
		return errors.New("migrator: applied migration number on db cannot be greater than the defined migration list")
	}

	if appliedCount == len(m.migrations) {
		m.logger.Printf("database schema up to date")
		return nil
	}

	// TODO count new pending?

	//for idx, migration := range m.migrations[appliedCount-1 : len(m.migrations)] {
	for idx, migration := range m.migrations[appliedCount:len(m.migrations)] {
		if err := m.migrate(idx+appliedCount, migration); err != nil {
			return errors.Wrapf(err, "migrator: error while running migration: %s", migration.String())
		}
	}

	m.logger.Printf("successfully applied all migrations!")

	return nil
}

func (m *Migrator) updateSchemaVersion(tx *sql.Tx, id int, version string) error {
	updateVersion := fmt.Sprintf("INSERT INTO %s (id, version, datetime) VALUES (%d, '%s', '%s')", m.tableName, id, version, time.Now().String())
	_, err := tx.Exec(updateVersion)
	if err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", version)
	}

	return nil
}

// readFile from embed.FS if provided or local fs as a fallback
func (m *Migrator) readFile(filename string) ([]byte, error) {
	if m.embedFS != nil {
		data, err := m.embedFS.ReadFile(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read initial schema file %q from embed.FS", filename)
		}

		return data, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read initial schema: %q", filename)
	}

	return data, nil
}

func (m *Migrator) migrateInitialSchema(migration *Migration) error {
	if migration.Name == "" {
		return errors.New("migration must have a name")
	}

	if migration.Run == nil && migration.RunTx == nil && migration.File == "" {
		return errors.New("migration must have a Run/RunTx function or a valid File path")
	}

	if migration.Run != nil && migration.File != "" {
		return errors.New("migration cannot have both Run function and File path")
	} else if migration.RunTx != nil && migration.File != "" {
		return errors.New("migration cannot have both RunTx function and File path")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	m.logger.Printf("applying base schema migration...")

	if migration.Run != nil {
		m.logger.Printf("applying base migration from Run: %q ...", migration.Name)

		if err = migration.Run(m.db); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.RunTx != nil {
		m.logger.Printf("applying base migration from RunTx: %q ...", migration.Name)

		if err = migration.RunTx(tx); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.File != "" {
		m.logger.Printf("applying base migration from file: %q %q ...", migration.Name, migration.File)

		// handle file based migration
		data, err := m.readFile(migration.File)
		if err != nil {
			return errors.Wrapf(err, "could not read migration from file: %q", migration.File)
		}

		if _, err = tx.Exec(string(data)); err != nil {
			return errors.Wrapf(err, "error applying schema migration from file: %q", migration.File)
		}
	}

	m.logger.Printf("applied base schema migration")

	for i, migrationItem := range m.migrations {
		if err = m.updateSchemaVersion(tx, i, migrationItem.Name); err != nil {
			return errors.Wrapf(err, "error updating migration versions: %s", "initial schema")
		}
	}

	m.logger.Printf("applied all schema migrations")

	return err
}

func (m *Migrator) migrateInitialSchemaOpt() error {
	if m.initialSchema == "" && m.initialSchemaFile != "" {
		data, err := m.readFile(m.initialSchemaFile)
		if err != nil {
			return errors.Wrapf(err, "could not read initial schema: %q", m.initialSchemaFile)
		}

		m.initialSchema = string(data)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				//err = fmt.Errorf("error rolling back: %s\n%s", errRb, err)
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	m.logger.Printf("applying base schema migration...")

	if _, err = tx.Exec(m.initialSchema); err != nil {
		return errors.Wrap(err, "error applying base schema migration")
	}

	if err = m.updateSchemaVersion(tx, 0, "initial schema"); err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", "initial schema")
	}

	//if len(m.migrations) > 0 {
	//	lastMigration := m.migrations[len(m.migrations)-1]
	//
	//	if err = m.updateVersion(tx, len(m.migrations), lastMigration.Name); err != nil {
	//		return errors.Wrapf(err, "error updating migration versions: %s", lastMigration.Name)
	//	}
	//}

	m.logger.Printf("applied base schema migration")

	return err
}

func (m *Migrator) migrate(migrationNumber int, migration *Migration) error {
	if migration.Name == "" {
		return errors.New("migration must have a name")
	}

	if migration.Run == nil && migration.RunTx == nil && migration.File == "" {
		return errors.New("migration must have a Run/RunTx function or a valid File path")
	}

	if migration.Run != nil && migration.File != "" {
		return errors.New("migration cannot have both Run function and File path")
	} else if migration.RunTx != nil && migration.File != "" {
		return errors.New("migration cannot have both RunTx function and File path")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				//err = fmt.Errorf("error rolling back: %s\n%s", errRb, err)
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	m.logger.Printf("applying migration: %q ...", migration.Name)

	if migration.Run != nil {
		if err = migration.Run(m.db); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.RunTx != nil {
		if err = migration.RunTx(tx); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.File != "" {
		m.logger.Printf("applying migration from file: %q %q ...", migration.Name, migration.File)

		// handle file based migration
		data, err := m.readFile(migration.File)
		if err != nil {
			return errors.Wrapf(err, "could not read migration from file: %q", migration.File)
		}

		if _, err = tx.Exec(string(data)); err != nil {
			return errors.Wrapf(err, "error applying schema migration from file: %q", migration.File)
		}
	}

	if err = m.updateSchemaVersion(tx, migrationNumber, migration.Name); err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", migration.Name)
	}

	m.logger.Printf("applied migration: %q", migration.Name)

	return err
}
