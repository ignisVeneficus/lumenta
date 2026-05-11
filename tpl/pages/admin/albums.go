package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/utils"
)

func AlbumsPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/albums", nil, nil)

		database := db.GetDatabase()
		albums, err := dao.QueryAlbum(database, ctx)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		albumList := make([]*adminData.AlbumViewTree, 0)
		for _, a := range albums {
			albumList = append(albumList, adminData.CreateAlbumView(a))
		}
		utils.SortByUint64Key(albumList, (*adminData.AlbumViewTree).GetSorting)
		forest := data.BuildForest(albumList)

		albumsCtx := adminData.AlbumsPageContext{}
		pageCtx := albumsCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "albums", tplData.SurfaceAdmin)
		albumsCtx.Breadcrumbs = tplData.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
			tplData.Breadcrumb{
				Label: i18n.T(loc, "nav.page.admin.albums.short", nil),
				Type:  "page",
			},
		}
		albumsCtx.Albums = *forest

		if err := r.RenderPage(c.Writer, "admin/albums", albumsCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
