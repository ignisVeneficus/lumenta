package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type mockToken struct {
	sub    string
	groups []string
	err    error
}

func (m mockToken) Claims(v interface{}) error {
	if m.err != nil {
		return m.err
	}

	raw := v.(*struct {
		Sub    string   `json:"sub"`
		Groups []string `json:"groups"`
	})

	raw.Sub = m.sub
	raw.Groups = m.groups
	return nil
}

type mockVerifier struct {
	token oidcToken
	err   error
}

func (m mockVerifier) Verify(ctx context.Context, raw string) (oidcToken, error) {
	return m.token, m.err
}

func TestOIDC_ContextFromRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("no token", func(t *testing.T) {
		o := &OIDCVerifier{
			verifier: mockVerifier{},
		}

		req := httptest.NewRequest("GET", "/", nil)

		res := o.ContextFromRequest(ctx, "", req)
		if res != nil {
			t.Fatalf("expected nil")
		}
	})

	t.Run("verify error", func(t *testing.T) {
		o := &OIDCVerifier{
			verifier: mockVerifier{
				err: errors.New("verify failed"),
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)
		if res != nil {
			t.Fatalf("expected nil")
		}
	})

	t.Run("claims error", func(t *testing.T) {
		o := &OIDCVerifier{
			verifier: mockVerifier{
				token: mockToken{
					err: errors.New("claims failed"),
				},
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)
		if res != nil {
			t.Fatalf("expected nil")
		}
	})

	t.Run("user role", func(t *testing.T) {
		o := &OIDCVerifier{
			AdminRole: "admin",
			verifier: mockVerifier{
				token: mockToken{
					sub:    "john",
					groups: []string{"user"},
				},
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)
		if res == nil {
			t.Fatalf("expected context")
		}

		if res.Role != dbo.RoleUser {
			t.Fatalf("expected user role")
		}
	})

	t.Run("admin role", func(t *testing.T) {
		o := &OIDCVerifier{
			AdminRole: "admin",
			verifier: mockVerifier{
				token: mockToken{
					sub:    "john",
					groups: []string{"user", "admin"},
				},
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)
		if res.Role != dbo.RoleAdmin {
			t.Fatalf("expected admin role")
		}
	})

	t.Run("admin role with spaces", func(t *testing.T) {
		o := &OIDCVerifier{
			AdminRole: "admin",
			verifier: mockVerifier{
				token: mockToken{
					sub:    "john",
					groups: []string{"  admin  "},
				},
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)
		if res.Role != dbo.RoleAdmin {
			t.Fatalf("expected admin role")
		}
	})

	t.Run("username and provider", func(t *testing.T) {
		o := &OIDCVerifier{
			verifier: mockVerifier{
				token: mockToken{
					sub: "john",
				},
			},
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer token")

		res := o.ContextFromRequest(ctx, "", req)

		if res.UserName == nil || *res.UserName != "john" {
			t.Fatalf("wrong username")
		}

		if res.Provider != authData.ProviderOIDC {
			t.Fatalf("wrong provider")
		}
	})
}
