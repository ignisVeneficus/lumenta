package public

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/utils"
)

func MainPage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/main", nil, nil)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		qty := 24
		hash := utils.ComputeDailyHash()

		images, err := dao.QueryImageRandomByACL(database, ctx, hash, acl.ACLContext, qty)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		makeURL := func(id routes.ImageID) string {
			return routes.CreateImagePath(id)
		}

		grid := data.ImageGrid{
			Images: grid.BuildGrid(images, cfg.Presentation.Grid, 1, makeURL),
		}
		albumCtx := data.MainPageContext{}
		pageCtx := albumCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "landing", data.SurfacePublic)
		albumCtx.ImageGrid = grid

		if err := r.RenderPage(c.Writer, "public/main", albumCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
