package auth

import "github.com/ignisVeneficus/lumenta/auth/data"

type AuthConfig struct {
	Mode    data.AuthProvider `yaml:"mode"`
	Forward AuthForward       `yaml:"forward"`
	OIDC    AuthOIDC          `yaml:"oidc"`
	JWT     JWTConfig         `yaml:"jwt"`
}

type AuthForward struct {
	UserHeader   string   `yaml:"user_header"`
	GroupsHeader string   `yaml:"groups_header"`
	TrustedCIDRs []string `yaml:"trusted_proxy_cidr"`
	AdminRole    string   `yaml:"admin_role"`
}

type AuthOIDC struct {
	Issuer    string `yaml:"issuer"`
	ClientID  string `yaml:"client_id"`
	AdminRole string `yaml:"admin_role"`
}
type JWTConfig struct {
	Secret string `yaml:"secret"`
}
