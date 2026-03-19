//go:build db_auth

package database

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

const (
	// defaultBcryptCost is the default cost factor for bcrypt hashing.
	defaultBcryptCost = 12

	// minPasswordLength is the minimum allowed password length.
	minPasswordLength = 8
)

// PasswordService handles password hashing and verification.
type PasswordService struct {
	cost int
}

// NewPasswordService creates a new PasswordService.
// Reads DB_AUTH_BCRYPT_COST from environment (default: 12).
func NewPasswordService() *PasswordService {
	cost := defaultBcryptCost
	if envCost := os.Getenv("DB_AUTH_BCRYPT_COST"); envCost != "" {
		if parsed, err := strconv.Atoi(envCost); err == nil && parsed >= bcrypt.MinCost && parsed <= bcrypt.MaxCost {
			cost = parsed
		}
	}
	return &PasswordService{cost: cost}
}

// HashPassword hashes a plaintext password using bcrypt.
// Returns the hashed password string or an error.
func (s *PasswordService) HashPassword(password string) (string, error) {
	if len(password) < minPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a bcrypt hash.
// Returns nil if the password matches, or an error if it doesn't.
func (s *PasswordService) VerifyPassword(hashedPassword, plainPassword string) error {
	if hashedPassword == "" {
		return fmt.Errorf("no password hash stored for this account")
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		return fmt.Errorf("invalid password")
	}
	return nil
}

// ValidatePasswordStrength checks if a password meets minimum requirements.
// Returns nil if valid, or an error describing the issue.
func (s *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	return nil
}
