// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/crypto/bcrypt"

	"github.com/autobrr/netronome/internal/config"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrUserAlreadyExists    = errors.New("username already exists")
)

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type UserService interface {
	CreateUser(ctx context.Context, username, password string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ValidatePassword(user *User, password string) bool
}

func (s *service) CreateUser(ctx context.Context, username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("%w: username and password required", ErrInvalidInput)
	}

	// Check if any user exists first
	count, err := s.count(ctx, "users", sq.Eq{})
	if err != nil && !isTableNotExistsError(err) {
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}

	if count > 0 {
		return nil, ErrRegistrationDisabled
	}

	// Check if username already exists
	exists, err := s.count(ctx, "users", sq.Eq{"username": username})
	if err != nil && !isTableNotExistsError(err) {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}

	if exists > 0 {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert user
	query := s.sqlBuilder.
		Insert("users").
		Columns("username", "password_hash").
		Values(username, string(hash))

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
	}

	var id int64
	if s.config.Type == config.Postgres {
		err = query.RunWith(tx).QueryRowContext(ctx).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		result, err := query.RunWith(tx).ExecContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		id, err = result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	// Disable registration
	var disableRegQuery string
	if s.config.Type == config.Postgres {
		disableRegQuery = `
			DELETE FROM registration_status;
			INSERT INTO registration_status (is_registration_enabled) VALUES (false);`
	} else {
		disableRegQuery = `
			INSERT INTO registration_status (is_registration_enabled) 
			VALUES (0) 
			ON CONFLICT (rowid) DO UPDATE SET is_registration_enabled = 0`
	}

	_, err = tx.ExecContext(ctx, disableRegQuery)
	if err != nil && !isTableNotExistsError(err) {
		return nil, fmt.Errorf("failed to disable registration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: string(hash),
	}, nil
}

func (s *service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("%w: username required", ErrInvalidInput)
	}

	query := s.sqlBuilder.
		Select("id", "username", "password_hash").
		From("users").
		Where(sq.Eq{"username": username})

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrUserNotFound
	}

	user := &User{}
	err = rows.Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning user: %w", err)
	}

	return user, nil
}

func (s *service) ValidatePassword(user *User, password string) bool {
	if user == nil || password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

func isTableNotExistsError(err error) bool {
	return err != nil && (err.Error() == "SQL logic error: no such table: users (1)" ||
		err.Error() == "SQL logic error: no such table: registration_status (1)")
}
