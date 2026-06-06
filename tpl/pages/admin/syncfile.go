package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/internal/locale"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func SyncFilePage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		syncFileIDStr := c.Param("id")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/sync_file/one", syncFileIDStr, map[string]any{
			"syncFile_id": syncFileIDStr,
		})

		syncFileId, err := tpl.ParseSyncFileID(syncFileIDStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		database := db.GetDatabase()
		syncFile, err := dao.GetSyncFileById(database, ctx, dbo.SyncFileID(syncFileId))
		if err != nil {
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "sync_file", routes.CreateAdminSyncFilesByPathPath(""), uint64(syncFileId))
			return
		}
		var ruleresult ruleengine.RuleResults
		if len(syncFile.RuleResultsJSON) > 0 {
			err := json.Unmarshal(syncFile.RuleResultsJSON, &ruleresult)
			if err != nil {
				logging.ExitErr(logScope, err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
		syncData := adminData.SyncFileData{
			SyncFile:   syncFile,
			RuleResult: ruleresult,
		}

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(),
			data.Breadcrumb{
				Link: tplData.Link{
					LabelKey: "nav.page.admin.sync_files.short",
					TitleKey: "nav.page.admin.sync_files.label",
					URL:      routes.CreateAdminSyncFilesPath(),
				},
				Type: "sync",
			},

			data.Breadcrumb{
				Link: tplData.Link{
					Label:    syncData.FullPathText(),
					URL:      routes.CreateAdminSyncFilesByPathPath(syncFile.PathFull()),
					TitleKey: "nav.page.admin.sync_file_history.label",
				},
				Type: "sync",
			},

			data.Breadcrumb{
				Link: tplData.Link{
					Label: locale.FormatTime(syncData.CreatedAt, loc),
				},
				Type: "file",
			},
		}

		syncCtx := adminData.SyncFilePageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Breadcrumbs = breadcrumbs
		syncCtx.File = syncData

		if err := r.RenderPage(c.Writer, "admin/syncfile", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
