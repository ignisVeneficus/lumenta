package server

import (
	"context"
	"net/http"

	"github.com/ignisVeneficus/lumenta/auth"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	authConfig "github.com/ignisVeneficus/lumenta/config/auth"
	"github.com/ignisVeneficus/lumenta/utils"

	"github.com/gin-gonic/gin"
)

type ExternalAuthRuntime interface {
	ContextFromRequest(ctx context.Context, ip string, request http.Request) *authData.ACLContext
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

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"access_token", // name
		token,          // value
		3600,           // maxAge (sec)
		"/",            // path
		"",             // domain
		true,           // secure (HTTPS)
		true,           // httpOnly
	)

	return true
}

func AuthContextMiddleware(ctx context.Context, cfg authConfig.AuthConfig, env config.Environment) gin.HandlerFunc {

	rt := GetAuthRuntime(ctx, cfg)

	JWT := NewJWTService(cfg.JWT.Secret)

	return func(c *gin.Context) {

		var ctx *authData.ACLContext
		var jwtCtx *authData.ACLContext
		var extCtx *authData.ACLContext

		extCtx = rt.ContextFromRequest(c, c.ClientIP(), *c.Request)

		if JWT != nil {
			token := auth.TokenForJWT(c.Request)
			jwtCtx = ContextFromToken(token, JWT)
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
			if ok := createJWTToken(c, JWT, *ctx); !ok {
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

func RequireRole(required authData.ACLRole) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx := auth.GetAuthContex(c)

		if ctx.Role.Compare(required) == -1 {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetAuthRuntime(ctx context.Context, cfg authConfig.AuthConfig) ExternalAuthRuntime {
	switch cfg.Mode {
	case authData.ProviderForward:
		return auth.ForwardVerifier{
			Cidrs:        cfg.Forward.NormalizedCIDRs,
			UserHeader:   cfg.Forward.UserHeader,
			GroupsHeader: cfg.Forward.GroupsHeader,
			AdminRole:    cfg.Forward.AdminRole,
		}

	case authData.ProviderOIDC:
		ret, err := auth.NewOIDCVerifier(ctx, cfg.OIDC.Issuer, cfg.OIDC.ClientID, cfg.OIDC.AdminRole)
		if err == nil {
			return ret
		}
	}
	return nil
}
