package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth/data"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type OIDCClaims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Groups  []string `json:"groups"`
}

type OIDCVerifier struct {
	verifier  oidcVerifier
	AdminRole string
}

type oidcToken interface {
	Claims(v interface{}) error
}

type oidcVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (oidcToken, error)
}

type realOIDCToken struct {
	t *oidc.IDToken
}

func (r realOIDCToken) Claims(v interface{}) error {
	return r.t.Claims(v)
}

type realVerifier struct {
	v *oidc.IDTokenVerifier
}

func (r realVerifier) Verify(ctx context.Context, raw string) (oidcToken, error) {
	tok, err := r.v.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	return realOIDCToken{tok}, nil
}

func NewOIDCVerifier(c context.Context, issuerURL, clientID, adminRole string) (*OIDCVerifier, error) {
	logg, ctx := logging.Enter(c, "auth/oidc/verifer", nil, map[string]any{"issuer": issuerURL, "client_id": clientID})
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

	return &OIDCVerifier{verifier: realVerifier{v}, AdminRole: adminRole}, nil
}

func (o *OIDCVerifier) verify(c context.Context, token string) (*OIDCClaims, error) {
	logg, ctx := logging.Enter(c, "auth.oidc.verif", nil, nil)
	if token == "" {
		err := errors.New("empty token")
		logging.ExitErr(logg, err)
		return nil, err
	}

	idToken, err := o.verifier.Verify(ctx, token)
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

func (o *OIDCVerifier) ContextFromRequest(c context.Context, ip string, request *http.Request) *authData.ACLContext {
	logg, ctx := logging.Enter(c, "auth/oidc/ctxFromRequest", nil, nil)
	token := TokenForOIDC(request)
	if token == "" {
		logging.Exit(logg, "NOT OK", map[string]any{"problem": "no token"})
		return nil
	}

	claims, err := o.verify(ctx, token)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil
	}

	role := dbo.RoleUser
	for _, g := range claims.Groups {
		if strings.TrimSpace(g) == o.AdminRole {
			role = dbo.RoleAdmin
		}
	}
	logging.Exit(logg, "OK", map[string]any{"role": role, "user": &claims.Subject})

	return &data.ACLContext{
		ACLContext: dbo.ACLContext{
			Role: role,
		},
		UserName: &claims.Subject,
		Provider: data.ProviderOIDC,
	}

}
