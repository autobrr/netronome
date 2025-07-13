// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be less than 72 characters")
	ErrInvalidSession   = errors.New("invalid session")
)

//func ValidatePassword(password string) error {
//	if len(password) < 8 {
//		return ErrPasswordTooShort
//	}
//	if len(password) > 72 {
//		return ErrPasswordTooLong
//	}
//	return nil
//}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MemoryOnlyPrefix is used to mark tokens that should only exist in memory
const MemoryOnlyPrefix = "mem_"

// isJWT checks if a token appears to be a JWT (has 3 base64-encoded parts separated by dots)
func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

func SignToken(token, secret string) string {
	if secret == "" {
		return MemoryOnlyPrefix + token
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	signature := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", token, signature)
}

func VerifyToken(signedToken, secret string) (string, error) {
	if strings.HasPrefix(signedToken, MemoryOnlyPrefix) {
		return strings.TrimPrefix(signedToken, MemoryOnlyPrefix), nil
	}

	if secret != "" {
		parts := strings.Split(signedToken, ".")
		
		var token, signature string
		
		// Handle different token formats
		if len(parts) == 2 {
			// Regular signed token: token.signature
			token, signature = parts[0], parts[1]
		} else if len(parts) == 4 {
			// Signed JWT token: header.payload.signature.our_signature
			jwtParts := strings.Join(parts[:3], ".")
			if isJWT(jwtParts) {
				token, signature = jwtParts, parts[3]
			} else {
				return "", fmt.Errorf("%w: malformed token", ErrInvalidSession)
			}
		} else {
			return "", fmt.Errorf("%w: malformed token", ErrInvalidSession)
		}

		expectedSignature := hmac.New(sha256.New, []byte(secret))
		expectedSignature.Write([]byte(token))
		expectedHex := hex.EncodeToString(expectedSignature.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedHex)) {
			return "", fmt.Errorf("%w: invalid signature", ErrInvalidSession)
		}

		return token, nil
	}

	return "", ErrInvalidSession
}
