package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/ignisVeneficus/lumenta/auth/data"
	authConfig "github.com/ignisVeneficus/lumenta/config/auth"
)

func ContextFromForwardAuth(headers http.Header, cfg authConfig.AuthForward) *data.ACLContext {

	user := headers.Get(cfg.UserHeader)
	if user == "" {
		return nil
	}

	role := data.RoleUser
	groups := strings.Split(headers.Get(cfg.GroupsHeader), ",")
	for _, g := range groups {
		if strings.TrimSpace(g) == cfg.AdminRole {
			role = data.RoleAdmin
		}
	}
	return &data.ACLContext{
		UserName: &user,
		Role:     role,
		Provider: data.ProviderForward,
	}
}

func ContextFromOIDC(ctx context.Context, token string, verifier OIDCVerifier, cfg authConfig.AuthOIDC) *data.ACLContext {

	claims, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil
	}

	role := data.RoleUser

	for _, g := range claims.Groups {
		if strings.TrimSpace(g) == cfg.AdminRole {
			role = data.RoleAdmin
		}
	}
	return &data.ACLContext{
		UserName: &claims.Subject,
		Role:     role,
		Provider: data.ProviderOIDC,
	}
}

func TokenFromRequest(r *http.Request) string {
	// Authorization: Bearer <token>
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(h, "Bearer ") {
			return strings.TrimPrefix(h, "Bearer ")
		}
	}

	// Cookie: access_token=<token>
	if c, err := r.Cookie("access_token"); err == nil {
		return c.Value
	}

	return ""
}

func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
