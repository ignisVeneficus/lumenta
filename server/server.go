package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
	"github.com/ignisVeneficus/lumenta/tpl/pages/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages/public"
)

var StaticRoot string = "web/static"

func Server(cfg config.Config) {
	ctx := context.Background()
	gin.SetMode(gin.ReleaseMode)

	if cfg.Env == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	}

	templatreResolver, err := tpl.NewTemplateResolver(ctx, "", tpl.DefaultFuncMap())
	if err != nil {
		panic(err)
	}

	r := gin.New()

	r.NoRoute(pages.Global404(templatreResolver, cfg))

	r.Use(
		RequestID(),
		Logger(),
		gin.Recovery(),
		AuthContextMiddleware(ctx, cfg.Auth, cfg.Env),
		SiteAccessMiddleware(cfg.Auth.GuestEnabled),
	)

	r.GET("/static/*filepath", func(c *gin.Context) {
		if cfg.Env != config.EnvDevelopment {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.FileFromFS(c.Param("filepath"), http.Dir(StaticRoot))
	})

	publicGrp := r.Group("/")
	{

		publicGrp.GET("/", public.MainPage(templatreResolver, cfg))
		/*
			web.GET("/album/:id", AlbumHandler)
			web.GET("/album/:aid/img/:iid", ImageHandler)
		*/
		publicGrp.GET(routes.GetTagsRootPath(), public.TagsRootPage(templatreResolver, cfg))
		publicGrp.GET(routes.GetTagPath(), public.TagPage(templatreResolver, cfg))

		/// "/img/:id/:type"
		publicGrp.GET(routes.GetImageDerivativePath(), DerivativeHandler(cfg))
	}
	adminGrp := r.Group("/admin")
	adminGrp.Use(RequireRole(authData.RoleAdmin))
	{
		adminGrp.GET("/", admin.MainPage(templatreResolver, cfg))
		adminGrp.GET(routes.GetAdminFsPath(), admin.FSPage(templatreResolver, cfg))

		adminGrp.GET(routes.GetAdminImgPath(), admin.ImagePage(templatreResolver, cfg))
		/*
			filesystem: /fs/
			Albums /album/:id
			Albums /album/new

			Albums list		GET /admin/albums
			New album form	GET /admin/albums/new
			save new album	POST /admin/albums
			Album edit		GET /admin/albums/:id
			Album edit		POST /admin/albums/:id
		*/
	}

	srv := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: r,

		ReadTimeout:       cfg.Server.Timeouts.Read,
		ReadHeaderTimeout: cfg.Server.Timeouts.Header,
		WriteTimeout:      cfg.Server.Timeouts.Write,
		IdleTimeout:       cfg.Server.Timeouts.Idle,
		//MaxHeaderBytes: cfg.Server.MaxHeaderBytes, // opcionális
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		// log fatal
	}

	r.Run(cfg.Server.Addr)

}
