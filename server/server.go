package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/api/endpoint"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
	"github.com/ignisVeneficus/lumenta/tpl/pages/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages/public"
)

var StaticRoot string = "web/static"

func Server(cfg config.Config, i18n *i18n.Service, ctx context.Context) {
	logScope, ctx := logging.Enter(ctx, "server/root", nil, nil)
	gin.SetMode(gin.ReleaseMode)

	if cfg.Env == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	}

	templatreResolver, err := tpl.NewTemplateResolver(ctx, "", i18n)
	if err != nil {
		logging.ExitErr(logScope, err)
		panic(err)
	}

	r := gin.New()

	r.NoRoute(pages.Global404(templatreResolver, cfg))

	r.Use(
		InputGuard(),
		RequestID(),
		Logger(),
		gin.Recovery(),
		AuthContextMiddleware(ctx, cfg.Site.BaseURL, cfg.Auth, cfg.Env),
		SiteAccessMiddleware(cfg.Auth.GuestEnabled),
		BrowserCache(),
	)

	r.GET("/static/*filepath", func(c *gin.Context) {
		if cfg.Env != config.EnvDevelopment {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.FileFromFS(c.Param("filepath"), http.Dir(StaticRoot))
	})

	publicGrp := r.Group("/")
	publicGrp.Use(DefaultHTMLMime())
	{

		publicGrp.GET("/", public.MainPage(templatreResolver, cfg, i18n))
		/*
			web.GET("/album/:id", AlbumHandler)
			web.GET("/album/:aid/img/:iid", ImageHandler)
		*/
		publicGrp.GET(routes.GetTagsRootPath(), public.TagsRootPage(templatreResolver, cfg, i18n))
		publicGrp.GET(routes.GetTagPath(), public.TagPage(templatreResolver, cfg, i18n))
		publicGrp.GET(routes.GetTagImagePath(), public.AlbumImagePage(templatreResolver, cfg, i18n))

		publicGrp.GET(routes.GetImagePath(), public.ImagePage(templatreResolver, cfg, i18n))

		publicGrp.GET(routes.GetAlbumPath(), public.AlbumPage(templatreResolver, cfg, i18n))
		publicGrp.GET(routes.GetAlbumsRootPath(), public.AlbumsRootPage(templatreResolver, cfg, i18n))
		publicGrp.GET(routes.GetAlbumImagePath(), public.AlbumImagePage(templatreResolver, cfg, i18n))

		/// "/img/:id/:type"
		publicGrp.GET(routes.GetImageDerivativePath(), DerivativeHandler(cfg))
	}
	adminGrp := r.Group("/admin")
	adminGrp.Use(RequireRole(dbo.RoleAdmin), DefaultHTMLMime())
	{
		adminGrp.GET("/", admin.MainPage(templatreResolver, cfg, i18n))
		adminGrp.GET(routes.GetAdminFsPath(), admin.FSPage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminImgPath(), admin.ImagePage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminAlbumsPath(), admin.AlbumsPage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminAlbumNewPath(), admin.NewAlbumPage(templatreResolver, cfg, i18n))
		adminGrp.POST(routes.GetAdminAlbumNewPath(), admin.NewAlbumPage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminAlbumPath(), admin.EditAlbumPage(templatreResolver, cfg, i18n))
		adminGrp.POST(routes.GetAdminAlbumPath(), admin.EditAlbumPage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminSyncRunsPath(), admin.SyncRunsListPage(templatreResolver, cfg, i18n))
		adminGrp.GET(routes.GetAdminSyncRunFilesPath(), admin.SyncRunFilesListPage(templatreResolver, cfg, i18n))

		adminGrp.GET(routes.GetAdminSyncFilesPath(), admin.SyncFilesListPage(templatreResolver, cfg, i18n))
		adminGrp.GET(routes.GetAdminSyncFilesByPathPath(), admin.SyncFilesListPathPage(templatreResolver, cfg, i18n))
		adminGrp.GET(routes.GetAdminSyncFilePath(), admin.SyncFilePage(templatreResolver, cfg, i18n))

		/*
			filesystem: /fs/
			Albums /album/:id
			Albums /album/new

			Albums list		GET /admin/albums
			New album form	GET /admin/albums/new
			save new album	POST /admin/albums/new
			Album edit		GET /admin/albums/:id
			Album edit		POST /admin/albums/:id
		*/
	}
	apiGrp := r.Group(routes.ApiPrefix)
	{
		apiGrp.GET(routes.GetApiTagCoordPath(), endpoint.ImageCoordByTags(cfg))
		apiGrp.GET(routes.GetApiAlbumsRootCoordPath(), endpoint.ImageCoordByAlbumRoot(cfg))
		apiGrp.GET(routes.GetApiAlbumCoordPath(), endpoint.ImageCoordByAlbums(cfg))
	}
	apiAdminGrp := apiGrp.Group(routes.AdminPrefix)
	apiAdminGrp.Use(RequireAPIRole(dbo.RoleAdmin))
	{
		// tags
		apiAdminGrp.GET(routes.GetApiAdminTagsPath(), endpoint.TagsQuery(cfg))
		// albums
		apiAdminGrp.GET(routes.GetApiAdminAlbumsPath(), endpoint.AlbumQuery(cfg))
		apiAdminGrp.PATCH(routes.GetApiAdminAlbumPath(), endpoint.AlbumPatch(cfg))
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
		logging.Panic(logScope, "start server", nil, err, "")
		panic(err)

	}

	err = r.Run(cfg.Server.Addr)
	logging.Return(logScope, err)
}
