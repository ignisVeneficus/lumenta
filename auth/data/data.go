package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type AuthProvider string

const (
	ProviderGuest   AuthProvider = "guest"
	ProviderJWT     AuthProvider = "jwt"
	ProviderOIDC    AuthProvider = "oidc"
	ProviderForward AuthProvider = "forward"
	ProviderDev     AuthProvider = "dev-environment"
)

type ACLContext struct {
	dbo.ACLContext
	UserName *string
	Provider AuthProvider
	Locale   string
}

func (ac ACLContext) L() string {
	return ac.Locale
}

var GuestName string = "Guest"

func GuestContext() *ACLContext {
	return &ACLContext{
		ACLContext: dbo.ACLContext{
			Role: dbo.RoleGuest,
		},
		Provider: ProviderGuest,
		UserName: &GuestName,
	}
}
