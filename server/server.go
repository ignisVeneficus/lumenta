package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
)

func Server(cfg config.Config) {
	gin.SetMode(gin.ReleaseMode)

	if cfg.Env == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	}

	runtimeCfg := AuthRuntime{
		JWT: NewJWTService(cfg.Auth.JWT.Secret),
	}

	r := gin.New()

	r.Use(
		RequestID(),
		Logger(),
		gin.Recovery(),
		AuthContextMiddleware(cfg.Auth, runtimeCfg, cfg.Env),
	)

	web := r.Group("/")
	{
		/*
			web.GET("/", HomeHandler)

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
