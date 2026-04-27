package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Run("valid password", func(t *testing.T) {
		hash, err := HashPassword("secret123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if hash == "" {
			t.Fatalf("expected hash, got empty string")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		_, err := HashPassword("")
		if err == nil {
			t.Fatalf("expected error for empty password")
		}
	})
}

func TestVerifyPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		if !VerifyPassword(hash, "secret123") {
			t.Fatalf("expected password to match")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		if VerifyPassword(hash, "wrong") {
			t.Fatalf("expected password to not match")
		}
	})

	t.Run("empty inputs", func(t *testing.T) {
		if VerifyPassword("", "secret") {
			t.Fatalf("expected false for empty hash")
		}
		if VerifyPassword(hash, "") {
			t.Fatalf("expected false for empty password")
		}
	})
}

func TestTokenForOIDC(t *testing.T) {
	t.Run("valid bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer abc123")

		token := TokenForOIDC(req)
		if token != "abc123" {
			t.Fatalf("expected abc123, got %s", token)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		token := TokenForOIDC(req)
		if token != "" {
			t.Fatalf("expected empty token, got %s", token)
		}
	})

	t.Run("invalid prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Basic xyz")

		token := TokenForOIDC(req)
		if token != "" {
			t.Fatalf("expected empty token, got %s", token)
		}
	})
}

func TestTokenForJWT(t *testing.T) {
	t.Run("header token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Auth-Token", "header-token")

		token := TokenForJWT(req)
		if token != "header-token" {
			t.Fatalf("expected header-token, got %s", token)
		}
	})

	t.Run("cookie token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "cookie-token",
		})

		token := TokenForJWT(req)
		if token != "cookie-token" {
			t.Fatalf("expected cookie-token, got %s", token)
		}
	})

	t.Run("header overrides cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Auth-Token", "header-token")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "cookie-token",
		})

		token := TokenForJWT(req)
		if token != "header-token" {
			t.Fatalf("expected header-token, got %s", token)
		}
	})

	t.Run("no token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		token := TokenForJWT(req)
		if token != "" {
			t.Fatalf("expected empty token, got %s", token)
		}
	})
}
