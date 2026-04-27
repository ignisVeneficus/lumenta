package auth

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("invalid CIDR %s: %v", cidr, err)
	}
	return n
}

func TestCIDRMatch(t *testing.T) {
	cidrs := []*net.IPNet{
		mustCIDR(t, "192.168.1.0/24"),
	}

	t.Run("match", func(t *testing.T) {
		if !CIDRMatch(cidrs, "192.168.1.10") {
			t.Fatalf("expected match")
		}
	})

	t.Run("no match", func(t *testing.T) {
		if CIDRMatch(cidrs, "10.0.0.1") {
			t.Fatalf("expected no match")
		}
	})

	t.Run("invalid ip", func(t *testing.T) {
		if CIDRMatch(cidrs, "not-an-ip") {
			t.Fatalf("expected false for invalid ip")
		}
	})
}

func TestForwardVerifier_ContextFromRequest(t *testing.T) {
	cidrs := []*net.IPNet{
		mustCIDR(t, "192.168.1.0/24"),
	}

	fv := ForwardVerifier{
		Cidrs:        cidrs,
		UserHeader:   "X-User",
		GroupsHeader: "X-Groups",
		AdminRole:    "admin",
	}

	ctx := context.Background()

	t.Run("valid user, normal role", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User", "john")
		req.Header.Set("X-Groups", "user,guest")

		res := fv.ContextFromRequest(ctx, "192.168.1.5", req)
		if res == nil {
			t.Fatalf("expected context, got nil")
		}

		if res.UserName == nil || *res.UserName != "john" {
			t.Fatalf("wrong username")
		}

		if res.Role != dbo.RoleUser {
			t.Fatalf("expected user role")
		}

		if res.Provider != authData.ProviderForward {
			t.Fatalf("wrong provider")
		}
	})

	t.Run("admin role detected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User", "adminuser")
		req.Header.Set("X-Groups", "user, admin ,other")

		res := fv.ContextFromRequest(ctx, "192.168.1.5", req)
		if res == nil {
			t.Fatalf("expected context, got nil")
		}

		if res.Role != dbo.RoleAdmin {
			t.Fatalf("expected admin role, got %v", res.Role)
		}
	})

	t.Run("ip not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User", "john")

		res := fv.ContextFromRequest(ctx, "10.0.0.1", req)
		if res != nil {
			t.Fatalf("expected nil for disallowed ip")
		}
	})

	t.Run("missing user header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		res := fv.ContextFromRequest(ctx, "192.168.1.5", req)
		if res != nil {
			t.Fatalf("expected nil when user header missing")
		}
	})

	t.Run("empty groups header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User", "john")

		res := fv.ContextFromRequest(ctx, "192.168.1.5", req)
		if res == nil {
			t.Fatalf("expected context")
		}

		if res.Role != dbo.RoleUser {
			t.Fatalf("expected default user role")
		}
	})
}
