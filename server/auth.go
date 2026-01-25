package server

import (
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/utils"

	"github.com/gin-gonic/gin"
)

const AuthContextKey = "auth"

type AuthRuntime struct {
	JWT  *JWTService
	OIDC auth.OIDCVerifier
}

func ContextFromToken(token string, jwtSvc *JWTService) auth.ACLContext {
	ctx := auth.GuestContext()

	if token == "" || jwtSvc == nil {
		return ctx
	}

	claims, err := jwtSvc.Verify(token)
	if err != nil {
		return ctx
	}

	ctx.UserID = &claims.UserID
	ctx.Role = auth.RoleUser
	ctx.Provider = auth.ProviderJWT

	return ctx
}

func AuthContextMiddleware(cfg config.AuthConfig, rt AuthRuntime, env config.Environment) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx := auth.GuestContext()

		switch cfg.Mode {
		case config.AuthModeForward:
			fwdCtx := auth.ContextFromForwardAuth(c.Request.Header, cfg.Forward)
			if fwdCtx.Provider != auth.ProviderGuest {
				ctx = fwdCtx
			}
		case config.AuthModeOIDC:
			if token := auth.BearerToken(c.Request); token != "" {
				oidcCtx := auth.ContextFromOIDC(c, token, rt.OIDC, cfg.OIDC)
				if oidcCtx.Provider != auth.ProviderGuest {
					ctx = oidcCtx
				}
			}
		}
		if ctx.Provider == auth.ProviderGuest && rt.JWT != nil {
			token := auth.TokenFromRequest(c.Request)
			jwtCtx := ContextFromToken(token, rt.JWT)
			c.Set(AuthContextKey, ctx)
			if jwtCtx.Provider != auth.ProviderGuest {
				ctx = jwtCtx
			}
		}

		if env == config.EnvDevelopment {
			ctx = auth.ACLContext{
				UserID:   utils.PtrUint64(uint64(1)),
				Role:     auth.RoleAdmin,
				Provider: auth.ProviderDev,
			}
		}

		c.Set(AuthContextKey, ctx)
		c.Next()
	}
}
