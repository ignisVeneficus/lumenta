package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/auth"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	authConfig "github.com/ignisVeneficus/lumenta/config/auth"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"

	"github.com/gin-gonic/gin"
)

func ContextFromToken(token string, jwtSvc *JWTService) *authData.ACLContext {
	if token == "" || jwtSvc == nil {
		return nil
	}

	claims, err := jwtSvc.Verify(token)
	if err != nil {
		return nil
	}
	userID := claims.UserID
	ctx := authData.ACLContext{
		ACLContext: dbo.ACLContext{
			ViewerUserID: (*dbo.UserID)(&userID),
			Role:         dbo.RoleUser,
		},
		UserName: &claims.Subject,
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

func SiteAccessMiddleware(EnableGuest bool) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx := auth.GetAuthContex(c)

		if ctx.Role == dbo.RoleGuest && !EnableGuest {

			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "authentication required",
				})
				c.Abort()
				return
			}

			next := url.QueryEscape(c.Request.URL.RequestURI())
			c.Redirect(http.StatusFound, "/login?next="+next)
			c.Abort()
			return
		}

		c.Next()
	}
}

func AuthContextMiddleware(ctx context.Context, issuer string, cfg authConfig.AuthConfig, env config.Environment) gin.HandlerFunc {

	rt := GetAuthRuntime(ctx, cfg)

	JWT := NewJWTService(cfg.JWT.Secret, WithIssuer(issuer))

	return func(c *gin.Context) {

		var ctx *authData.ACLContext
		var jwtCtx *authData.ACLContext
		var extCtx *authData.ACLContext

		extCtx = rt.ContextFromRequest(c, c.ClientIP(), c.Request)

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
			ctx.ViewerUserID = jwtCtx.ViewerUserID
		// only external
		case jwtCtx == nil && extCtx != nil:
			//TODO: db lookup
			ctx = extCtx
			if ok := createJWTToken(c, JWT, *ctx); !ok {
				return
			}

		//only internal
		case jwtCtx != nil:
			ctx = jwtCtx
			ctx.Role = dbo.RoleUser
		// guest
		default:
			ctx = authData.GuestContext()
		}

		if env == config.EnvDevelopment {
			userID := dbo.UserID(1)
			ctx = &authData.ACLContext{
				ACLContext: dbo.ACLContext{
					ViewerUserID: &userID,
					Role:         dbo.RoleAdmin,
				},
				UserName: utils.PtrString("dev admin"),
				Provider: authData.ProviderDev,
				Locale:   "en",
				//Locale: "hu",
			}
		}

		auth.SetAuthContex(c, *ctx)
		c.Next()
	}
}

func RequireAPIRole(required dbo.ACLRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := auth.GetAuthContex(c)
		if ctx.Role.Compare(required) == -1 {
			c.AbortWithStatusJSON(http.StatusForbidden,
				data.CreateError("admin access required"))
			return
		}
		c.Next()
	}
}

func RequireRole(required dbo.ACLRole) gin.HandlerFunc {
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

func GetAuthRuntime(ctx context.Context, cfg authConfig.AuthConfig) auth.ExternalAuthRuntime {
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
