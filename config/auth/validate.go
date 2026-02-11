package auth

import (
	"errors"
	"fmt"
	"net"

	"github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (a AuthConfig) Validate(v *validate.ValidationErrors, path string) {
	switch a.Mode {
	case data.ProviderForward:
		validate.LogConfigOK(path+"/mode", a.Mode)
		validate.RequireString(v, path+"/forward/user_header", a.Forward.UserHeader)

		ok := false
		for i, c := range a.Forward.TrustedCIDRs {
			p := fmt.Sprintf("%s/forward/trusted_proxy_cidr[%d]", path, i)
			_, _, err := net.ParseCIDR(c)
			if err != nil {
				validate.LogConfigError(p, c, err)
				v.Add(err)
			} else {
				validate.LogConfigOK(p, c)
				ok = true
			}
		}
		if !ok {
			err := errors.New("trusted_proxy_cidr must not be empty")
			validate.LogConfigError(path+"/forward/trusted_proxy_cidr", a.Forward.TrustedCIDRs, err)
			v.Add(err)
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
	validate.RequireString(v, path+"/jwt/secret", a.JWT.Secret)

}
