package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/ignisVeneficus/lumenta/auth/data"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/logging"
)

type OIDCClaims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Groups  []string `json:"groups"`
}

type OIDCVerifier struct {
	verifier  *oidc.IDTokenVerifier
	AdminRole string
}

func NewOIDCVerifier(ctx context.Context, issuerURL, clientID, adminRole string) (*OIDCVerifier, error) {
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

	return &OIDCVerifier{verifier: v, AdminRole: adminRole}, nil
}

func (o *OIDCVerifier) verify(ctx context.Context, token string) (*OIDCClaims, error) {
	logg := logging.Enter(ctx, "auth.oidc.verif", nil)
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

func (o *OIDCVerifier) ContextFromRequest(ctx context.Context, ip string, request http.Request) *authData.ACLContext {
	logg := logging.Enter(ctx, "auth.oidc.ctxFromRequest", nil)
	token := TokenForOIDC(&request)
	if token == "" {
		logging.Exit(logg, "NOT OK", map[string]any{"problem": "no token"})
		return nil
	}

	claims, err := o.verify(ctx, token)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil
	}

	role := data.RoleUser
	for _, g := range claims.Groups {
		if strings.TrimSpace(g) == o.AdminRole {
			role = data.RoleAdmin
		}
	}
	logging.Exit(logg, "OK", map[string]any{"role": role, "user": &claims.Subject})
	return &data.ACLContext{
		UserName: &claims.Subject,
		Role:     role,
		Provider: data.ProviderOIDC,
	}

}
