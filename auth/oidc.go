package auth

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/ignisVeneficus/lumenta/logging"
)

type OIDCClaims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Groups  []string `json:"groups"`
}

type OIDCVerifier interface {
	Verify(ctx context.Context, token string) (*OIDCClaims, error)
}

type OIDCVerifierImpl struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCVerifier(ctx context.Context, issuerURL, clientID string) (*OIDCVerifierImpl, error) {
	logg := logging.Enter(ctx, "auth.oidc.verifer", map[string]any{"issuer": issuerURL, "client_id": clientID})
	if issuerURL == "" || clientID == "" {
		err := errors.New("oidc issuer/client_id is empty")
		logging.ExitErr(logg, err)
		return nil, err
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}

	v := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})
	logging.Exit(logg, "ok", nil)

	return &OIDCVerifierImpl{verifier: v}, nil
}

func (o *OIDCVerifierImpl) Verify(ctx context.Context, token string) (*OIDCClaims, error) {
	logg := logging.Enter(ctx, "auth.oidc.verifer.verif", nil)
	if token == "" {
		err := errors.New("empty token")
		logging.ExitErr(logg, err)
		return nil, err
	}

	idToken, err := o.verifier.Verify(context.Background(), token)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}

	var raw struct {
		Sub    string   `json:"sub"`
		Groups []string `json:"groups"`
	}

	if err := idToken.Claims(&raw); err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	ret := &OIDCClaims{
		Subject: raw.Sub,
		Groups:  raw.Groups,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil
}
