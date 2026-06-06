package public

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
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

		daily := make([]presentation.TagMeaning, 0)
		for _, tm := range presentation.TagMeaningList {
			if cfg.Presentation.TagMeaningConfig.MeaningMap.HasFeature(tm, presentation.TagFeatureDaily) {
				daily = append(daily, tm)
			}
		}
		logging.Debug(logScope, "daily", map[string]any{
			"daily": daily,
		})
		dailyTags := make([]data.DailyTag, 0)
		if len(daily) > 0 {
			/* tags, daily */
			tags, err := dao.QueryTagsByACL(database, ctx, acl.ACLContext)
			if err != nil {
				logging.ExitErr(logScope, err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			tagsPoi := dbo.TagsWCountToPointer(tags)
			tagDiscovery := data.CreateTagDiscovery(cfg.Presentation.TagMeaningConfig, tagsPoi)
			logging.Debug(logScope, "discovery", map[string]any{
				"discovery": tagDiscovery,
			})

			for _, tm := range daily {
				itm := tagDiscovery.GetRandom(tm, hash)
				if itm != nil {
					dailyTags = append(dailyTags, data.DailyTag{
						Type: string(tm),
						Name: itm.Name,
						Url:  routes.CreateTagPath(routes.TagID(itm.ID)),
					})
				}
			}
		}

		albumCtx := data.MainPageContext{}
		pageCtx := albumCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "landing", data.SurfacePublic)
		albumCtx.ImageGrid = grid
		albumCtx.DailyTags = dailyTags

		if err := r.RenderPage(c.Writer, "public/main", albumCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
