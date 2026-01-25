package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

const (
	HeaderAdminGroup string = "lumenta-admin"
)

type ACLRole string

const (
	RoleGuest ACLRole = "guest"
	RoleUser  ACLRole = "user"
	RoleAdmin ACLRole = "admin"
)

type AuthProvider string

const (
	ProviderGuest   AuthProvider = "guest"
	ProviderJWT     AuthProvider = "jwt"
	ProviderOIDC    AuthProvider = "oidc"
	ProviderForward AuthProvider = "forward-auth"
	ProviderDev     AuthProvider = "dev-environment"
)

type ACLContext struct {
	UserID   *uint64
	Role     ACLRole
	Provider AuthProvider
}

func (a ACLContext) IsAnyUser() bool {
	return a.UserID != nil
}

func (a ACLContext) IsAdmin() bool {
	return a.Role == "admin"
}

func (a ACLContext) AsParamArray() []any {
	return []any{a.IsAnyUser(), a.UserID, a.IsAdmin()}
}
func (a *ACLContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("role", string(a.Role))
		logging.Uint64If(e, "userID", a.UserID)
	}
}

func GuestContext() ACLContext {
	return ACLContext{
		Role:     RoleGuest,
		Provider: ProviderGuest,
	}
}

func ContextFromForwardAuth(headers http.Header, cfg config.AuthForward) ACLContext {

	user := headers.Get(cfg.UserHeader)
	if user == "" {
		return GuestContext()
	}

	uid, err := strconv.ParseUint(user, 10, 64)
	if err != nil {
		return GuestContext()
	}

	role := RoleUser
	groups := strings.Split(headers.Get(cfg.GroupsHeader), ",")
	for _, g := range groups {
		if strings.TrimSpace(g) == cfg.AdminRole {
			role = RoleAdmin
		}
	}

	return ACLContext{
		UserID:   &uid,
		Role:     role,
		Provider: ProviderForward,
	}
}

func ContextFromOIDC(ctx context.Context, token string, verifier OIDCVerifier, cfg config.AuthOIDC) ACLContext {

	claims, err := verifier.Verify(ctx, token)
	if err != nil {
		return GuestContext()
	}

	uid, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return GuestContext()
	}

	role := RoleUser

	for _, g := range claims.Groups {
		if strings.TrimSpace(g) == cfg.AdminRole {
			role = RoleAdmin
		}
	}
	return ACLContext{
		UserID:   &uid,
		Role:     role,
		Provider: ProviderOIDC,
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
