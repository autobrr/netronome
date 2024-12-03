// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
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
	// Check if any user exists first
	var count int
	query, args, err := s.sqlBuilder.Select("COUNT(*)").From("users").ToSql()
	if err != nil {
		return nil, err
	}
	err = s.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		return nil, err
	}

	if err == nil && count > 0 {
		return nil, ErrRegistrationDisabled
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var insertQuery string
	var queryArgs []interface{}

	if s.config.Type == Postgres {
		insertQuery, queryArgs, err = s.sqlBuilder.
			Insert("users").
			Columns("username", "password_hash").
			Values(username, string(hash)).
			Suffix("RETURNING id").
			ToSql()
	} else {
		insertQuery, queryArgs, err = s.sqlBuilder.
			Insert("users").
			Columns("username", "password_hash").
			Values(username, string(hash)).
			ToSql()
	}
	if err != nil {
		return nil, err
	}

	var id int64
	if s.config.Type == Postgres {
		err = tx.QueryRowContext(ctx, insertQuery, queryArgs...).Scan(&id)
		if err != nil {
			return nil, err
		}
	} else {
		result, err := tx.ExecContext(ctx, insertQuery, queryArgs...)
		if err != nil {
			return nil, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	// ensuring registration is disabled
	var disableRegQuery string
	if s.config.Type == Postgres {
		// For Postgres, use DELETE + INSERT to ensure only one row exists
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
	if err != nil {
		if !isTableNotExistsError(err) {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: string(hash),
	}, nil
}

func (s *service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query, args, err := s.sqlBuilder.
		Select("id", "username", "password_hash").
		From("users").
		Where("username = ?", username).
		ToSql()
	if err != nil {
		return nil, err
	}

	user := &User{}
	err = s.db.QueryRowContext(ctx, query, args...).
		Scan(&user.ID, &user.Username, &user.PasswordHash)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) ValidatePassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

func isTableNotExistsError(err error) bool {
	return err != nil && (err.Error() == "SQL logic error: no such table: users (1)" ||
		err.Error() == "SQL logic error: no such table: registration_status (1)")
}
