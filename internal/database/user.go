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

// Add these methods to the Service interface in database.go
type UserService interface {
	CreateUser(ctx context.Context, username, password string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ValidatePassword(user *User, password string) bool
}

func (s *service) CreateUser(ctx context.Context, username, password string) (*User, error) {
	// Check if any user exists first
	var count int
	err := s.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		return nil, err
	}

	if err == nil && count > 0 {
		return nil, ErrRegistrationDisabled
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create user
	result, err := tx.ExecContext(ctx,
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, string(hash),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// After creating the first user, ensure registration is disabled
	_, err = tx.ExecContext(ctx, `
		INSERT INTO registration_status (is_registration_enabled) 
		VALUES (0) 
		ON CONFLICT (rowid) DO UPDATE SET is_registration_enabled = 0
	`)
	if err != nil {
		// If the table doesn't exist, that's fine - the migration will create it
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
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, username, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash)

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
