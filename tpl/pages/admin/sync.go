package admin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
)

const (
	syncRunPerPage uint64 = 20
)

func SyncRunsListPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		pageStr := c.DefaultQuery(data.SyncPageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/sync_run", nil, map[string]any{
			"page": pageStr,
		})

		page, err := tpl.ParsePaging(pageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		url := routes.BuildAdminSyncRunListPath()
		url.WithUintQuery(data.SyncPageParam, page)

		database := db.GetDatabase()
		syncs, err := dao.QuerySyncRunPaged(database, ctx, (page-1)*syncRunPerPage, syncRunPerPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		syncData := make([]adminData.SyncRunData, len(syncs))
		for i, s := range syncs {
			syncData[i] = adminData.SyncRunData{
				SyncRun: s,
			}
		}
		count, err := dao.CountSyncRun(database, c)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
			data.Breadcrumb{
				Label: i18n.T(loc, "nav.page.admin.sync_runs.short", nil),
				Type:  "sync",
			},
		}

		paging := data.CreatePaging(*url, data.SyncPageParam, page, count, syncRunPerPage)

		syncCtx := adminData.SyncRunsPageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Paging = paging
		syncCtx.SyncRuns = syncData
		syncCtx.Breadcrumbs = breadcrumbs

		if err := r.RenderPage(c.Writer, "admin/syncruns", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
