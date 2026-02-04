package auth

import (
	"errors"

	"github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (a AuthConfig) Validate(v *validate.ValidationErrors, path string) {
	switch a.Mode {
	case data.ProviderForward:
		validate.LogConfigOK(path+"/mode", a.Mode)
		validate.RequireString(v, path+"/forward/user_header", a.Forward.UserHeader)

		if len(a.Forward.TrustedCIDRs) == 0 {
			err := errors.New("trusted_proxy_cidr must not be empty")
			validate.LogConfigError(path+"/forward/trusted_proxy_cidr", a.Forward.TrustedCIDRs, err)
			v.Add(err)
		} else {
			validate.LogConfigOK(path+"/forward/trusted_proxy_cidr", a.Forward.TrustedCIDRs)
		}
		validate.RequireString(v, path+"/forward/admin_role", a.Forward.AdminRole)
	case data.ProviderOIDC:
		validate.LogConfigOK(path+"/mode", a.Mode)
		validate.RequireString(v, path+"/oidc/issuer", a.OIDC.Issuer)
		validate.RequireString(v, path+"/oidc/client_id", a.OIDC.ClientID)
		validate.RequireString(v, path+"/oidc/admin_role", a.OIDC.AdminRole)
	default:
		err := errors.New("mode must be forward or oidc")
		validate.LogConfigError(path+"/mode", a.Mode, err)
		v.Add(err)
	}
}
