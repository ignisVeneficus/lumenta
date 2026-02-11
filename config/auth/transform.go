package auth

import (
	"net"

	"github.com/ignisVeneficus/lumenta/auth/data"
)

func (ac *AuthConfig) TransformAfterValidation() error {
	switch ac.Mode {
	case data.ProviderForward:
		ac.Forward.TransformAfterValidation()
	}
	return nil
}

func (fc *AuthForward) TransformAfterValidation() error {
	fc.NormalizedCIDRs = []*net.IPNet{}
	for _, c := range fc.TrustedCIDRs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		fc.NormalizedCIDRs = append(fc.NormalizedCIDRs, n)
	}
	return nil
}
