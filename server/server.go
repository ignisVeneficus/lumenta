package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

var StaticRoot string = "web/static"

func Server(cfg config.Config) {
	cfx := context.Background()
	gin.SetMode(gin.ReleaseMode)

	if cfg.Env == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	}

	runtimeCfg := AuthRuntime{
		JWT: NewJWTService(cfg.Auth.JWT.Secret),
	}
	templatreResolver, err := tpl.NewTemplateResolver(cfx, "", tpl.DefaultFuncMap())
	if err != nil {
		panic(err)
	}

	r := gin.New()

	r.Use(
		RequestID(),
		Logger(),
		gin.Recovery(),
		AuthContextMiddleware(cfg.Auth, runtimeCfg, cfg.Env),
	)

	r.GET("/static/*filepath", func(c *gin.Context) {
		if cfg.Env != config.EnvDevelopment {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.FileFromFS(c.Param("filepath"), http.Dir(StaticRoot))
	})

	web := r.Group("/")
	{

		web.GET("/", pages.MainPage(templatreResolver, cfg))
		/*
			web.GET("/album/:id", AlbumHandler)
			web.GET("/album/:aid/img/:iid", ImageHandler)
		*/
		web.GET("/img/:id/:type", DerivativeHandler(cfg))
	}
	/*
		admin := r.Group("/admin")
		admin.Use(RequireRole(RoleAdmin))
		{
			admin.GET("/", AdminDashboard)
		}
	*/
	r.Run(cfg.Server.Addr)

}
