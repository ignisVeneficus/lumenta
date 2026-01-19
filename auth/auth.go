package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is a balanced default for interactive logins
	DefaultBcryptCost = bcrypt.DefaultCost // jelenleg 10
)

// HashPassword hashes a plaintext password using bcrypt.
// The returned value is safe to store in the database.
func HashPassword(plain string) (string, error) {
	if len(plain) == 0 {
		return "", errors.New("password must not be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), DefaultBcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a bcrypt hash with a plaintext password.
// Returns true if they match.
func VerifyPassword(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
