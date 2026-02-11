package auth

import (
	"errors"
	"net/http"
	"strings"

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

func TokenForOIDC(r *http.Request) string {
	// Authorization: Bearer <token>
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(h, "Bearer ") {
			return strings.TrimPrefix(h, "Bearer ")
		}
	}
	return ""
}

func TokenForJWT(r *http.Request) string {
	if h := r.Header.Get("X-Auth-Token"); h != "" {
		return h
	}
	// Cookie: access_token=<token>
	if c, err := r.Cookie("access_token"); err == nil {
		return c.Value
	}

	return ""
}
