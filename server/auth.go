package server

import (
	"github.com/ignisVeneficus/lumenta/auth"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	authConfig "github.com/ignisVeneficus/lumenta/config/auth"
	"github.com/ignisVeneficus/lumenta/utils"

	"github.com/gin-gonic/gin"
)

type AuthRuntime struct {
	JWT  *JWTService
	OIDC auth.OIDCVerifier
}

func ContextFromToken(token string, jwtSvc *JWTService) *authData.ACLContext {
	if token == "" || jwtSvc == nil {
		return nil
	}

	claims, err := jwtSvc.Verify(token)
	if err != nil {
		return nil
	}
	ctx := authData.ACLContext{
		UserID:   &claims.UserID,
		UserName: &claims.Subject,
		Role:     authData.RoleUser,
		Provider: authData.ProviderJWT,
	}

	return &ctx
}
func createJWTToken(c *gin.Context, jwtSvc *JWTService, acl authData.ACLContext) bool {
	token, err := jwtSvc.Issue(acl)
	if err != nil {
		c.AbortWithStatus(500)
		return false
	}
	c.Header("X-Auth-Token", token)
	return true
}

func AuthContextMiddleware(cfg authConfig.AuthConfig, rt AuthRuntime, env config.Environment) gin.HandlerFunc {
	return func(c *gin.Context) {

		var ctx *authData.ACLContext
		var jwtCtx *authData.ACLContext
		var extCtx *authData.ACLContext

		switch cfg.Mode {
		case authData.ProviderForward:
			extCtx = auth.ContextFromForwardAuth(c.Request.Header, cfg.Forward)
		case authData.ProviderOIDC:
			if token := auth.BearerToken(c.Request); token != "" {
				extCtx = auth.ContextFromOIDC(c, token, rt.OIDC, cfg.OIDC)
			}
		}
		if rt.JWT != nil {
			token := auth.TokenFromRequest(c.Request)
			jwtCtx = ContextFromToken(token, rt.JWT)
		}
		if jwtCtx != nil && extCtx != nil {
			if (*jwtCtx.UserName) != (*extCtx.UserName) {
				jwtCtx = nil
			}
		}
		switch {
		//use external and jwt
		case jwtCtx != nil && extCtx != nil:
			ctx = extCtx
			ctx.UserID = jwtCtx.UserID
		// only external
		case jwtCtx == nil && extCtx != nil:
			//TODO db lookup
			ctx = extCtx
			if ok := createJWTToken(c, rt.JWT, *ctx); !ok {
				return
			}

		//only internal
		case jwtCtx != nil:
			ctx = jwtCtx
			ctx.Role = authData.RoleUser
		// guest
		default:
			ctx = authData.GuestContext()
		}

		if env == config.EnvDevelopment {
			ctx = &authData.ACLContext{
				UserID:   utils.PtrUint64(uint64(1)),
				UserName: utils.PtrString("dev admin"),
				Role:     authData.RoleAdmin,
				Provider: authData.ProviderDev,
			}
		}

		auth.SetAuthContex(c, *ctx)
		c.Next()
	}
}
