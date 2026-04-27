package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/internal/locale"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func SyncFilePage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		syncFileIDStr := c.Param("id")
		logg := logging.Enter(c, "page.admin.syncRun.one", map[string]any{
			"syncFile_id": syncFileIDStr,
		})

		syncFileId, err := tpl.ParseID(syncFileIDStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		database := db.GetDatabase()
		syncFile, err := dao.GetSyncFileById(database, c, syncFileId)
		if err != nil {
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "sync_file", routes.CreateAdminSyncFilesByPathPath(""), syncFileId)
			return
		}
		var ruleresult ruleengine.RuleResults
		if len(syncFile.RuleResultsJSON) > 0 {
			err := json.Unmarshal(syncFile.RuleResultsJSON, &ruleresult)
			if err != nil {
				logging.ExitErr(logg, err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
		syncData := adminData.SyncFileData{
			SyncFile:   syncFile,
			RuleResult: ruleresult,
		}

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
			data.Breadcrumb{
				Label: i18n.T(loc, "nav.page.admin.sync_files.short", nil),
				Type:  "sync",
				Title: i18n.T(loc, "nav.page.admin.sync_files.label", nil),
				Link:  template.URL(routes.CreateAdminSyncFilesPath()),
			},

			data.Breadcrumb{
				Label: syncData.FullPathText(),
				Type:  "sync",
				Link:  template.URL(routes.CreateAdminSyncFilesByPathPath(syncFile.PathFull())),
				Title: i18n.T(loc, "nav.page.admin.sync_file_history.label", nil),
			},

			data.Breadcrumb{
				Label: locale.FormatTime(syncData.CreatedAt, loc),
				Type:  "file",
			},
		}

		syncCtx := adminData.SyncFilePageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Breadcrumbs = breadcrumbs
		syncCtx.File = syncData

		if err := r.RenderPage(c.Writer, "admin/syncfile", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logg, err)
			return
		}
		logging.Exit(logg, "ok", nil)
	}
}
