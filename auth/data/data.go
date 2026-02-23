package data

import (
	"fmt"
	"strings"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

type ACLRole string

func (r ACLRole) Compare(role ACLRole) int {
	if r == role {
		return 0
	}
	switch r {
	case RoleGuest:
		return -1
	case RoleAdmin:
		return 1
	}
	if role == RoleGuest {
		return 1
	}
	return -1
}

const (
	RoleGuest ACLRole = "guest"
	RoleUser  ACLRole = "user"
	RoleAdmin ACLRole = "admin"
)

func ParseRole(s string) (ACLRole, error) {
	switch strings.ToLower(s) {
	case "guest":
		return RoleGuest, nil
	case "user":
		return RoleUser, nil
	case "admin":
		return RoleAdmin, nil
	default:
		return "", fmt.Errorf("invalid role: %s", s)
	}
}
func IsValidRole(s ACLRole) bool {
	switch strings.ToLower(string(s)) {
	case string(RoleGuest), string(RoleUser), string(RoleAdmin):
		return true
	default:
		return false
	}
}

type AuthProvider string

const (
	ProviderGuest   AuthProvider = "guest"
	ProviderJWT     AuthProvider = "jwt"
	ProviderOIDC    AuthProvider = "oidc"
	ProviderForward AuthProvider = "forward"
	ProviderDev     AuthProvider = "dev-environment"
)

type ACLContext struct {
	UserID   *uint64
	UserName *string
	Role     ACLRole
	Provider AuthProvider
}

var GuestName string = "Guest"

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

func GuestContext() *ACLContext {
	return &ACLContext{
		Role:     RoleGuest,
		Provider: ProviderGuest,
		UserName: &GuestName,
	}
}
