package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

const (
	syncFilePerPage uint64 = 50
)

/*
sync id -> c.Param
search -> query
filename -> path, mint a fs eseten

*/

func parsePath(p string) (root, path, filename, ext string) {
	if p == "" {
		return "", "", "", ""
	}
	p = strings.TrimPrefix(p, "/")
	parts := strings.Split(p, "/")
	root = parts[0]
	if len(parts) == 1 {
		return root, "", "", ""
	}
	file := parts[len(parts)-1]

	if len(parts) > 2 {
		path = strings.Join(parts[1:len(parts)-1], "/")
	}

	dot := strings.LastIndex(file, ".")
	if dot > 0 && dot < len(file)-1 {
		filename = file[:dot]
		ext = file[dot+1:]
	} else {
		filename = file
		ext = ""
	}
	return
}

/*
Sync files for one image
*/
func SyncFilesListPathPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		fPath := c.Param("fPath")
		pageStr := c.DefaultQuery(routes.SyncPageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/sync_file/path", fPath, map[string]any{
			"page": pageStr,
			"path": fPath,
		})

		root, path, filename, ext := parsePath(fPath)

		page, err := tpl.ParsePaging(pageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		url := routes.BuildAdminSyncFilesByPathPath(dbo.BuildFullPath(root, path, filename, ext))
		url.WithUintQuery(routes.SyncPageParam, page)

		database := db.GetDatabase()

		syncs, err := dao.QuerySyncFileByFilePathPaged(database, ctx, root, path, filename, ext, (page-1)*syncFilePerPage, syncFilePerPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		count, err := dao.CountSyncFileByPath(database, ctx, root, path, filename, ext)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		syncData := make([]adminData.SyncFileData, len(syncs))
		for i, s := range syncs {
			syncData[i] = adminData.SyncFileData{
				SyncFile: s,
			}
		}

		paging := data.CreatePaging(*url, routes.SyncPageParam, page, count, syncFilePerPage)

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
		}
		rootElem := data.Breadcrumb{
			Label: i18n.T(loc, "nav.page.admin.sync_files.short", nil),
			Type:  "sync",
		}
		if root != "" {
			rootElem.Link = template.URL(routes.CreateAdminSyncFilesPath())
			rootElem.Title = i18n.T(loc, "nav.page.admin.sync_files.label", nil)

			breadcrumbs = append(breadcrumbs,
				rootElem,
				data.Breadcrumb{
					Label: fPath,
					Type:  "sync",
				})
		} else {
			breadcrumbs = append(breadcrumbs, rootElem)
		}

		syncCtx := adminData.SyncFilesPageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Paging = paging
		syncCtx.SyncFiles = syncData
		syncCtx.Breadcrumbs = breadcrumbs
		syncCtx.HasSearch = false
		syncCtx.HasFileButton = false

		if err := r.RenderPage(c.Writer, "admin/syncfiles", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}

/*
files for one run
*/
func SyncRunFilesListPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		syncRunIDStr := c.Param("id")
		filterIn := c.QueryArray(routes.FilterParam)
		pageStr := c.DefaultQuery(routes.SyncPageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/sync_file/sync_run", syncRunIDStr, map[string]any{
			"page":       pageStr,
			"syncRun_id": syncRunIDStr,
			"filter":     filterIn,
		})

		filter := make([]string, 0, len(filterIn))
		for _, f := range filterIn {
			if f != "" {
				filter = append(filter, f)
			}
		}

		syncRunId, err := tpl.ParseSyncRunID(syncRunIDStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		page, err := tpl.ParsePaging(pageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		url := routes.BuildAdminSyncRunFilesPath(syncRunId)
		url.WithUintQuery(routes.SyncPageParam, page)
		if len(filter) > 0 {
			url.WithArrayParams(routes.FilterParam, filter)
		}

		database := db.GetDatabase()
		dbSyncRunID := dbo.SyncRunID(syncRunId)
		_, err = dao.GetSyncRunByID(database, ctx, dbSyncRunID)
		if err != nil {
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "sync_run", routes.CreateAdminSyncRunListPath(), uint64(syncRunId))
			return
		}

		syncs, err := dao.QuerySyncFileBySyncIDPaged(database, ctx, dbSyncRunID, filter, (page-1)*syncFilePerPage, syncFilePerPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		count, err := dao.CountSyncFileBySyncID(database, ctx, dbSyncRunID, filter)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		syncData := make([]adminData.SyncFileData, len(syncs))
		for i, s := range syncs {
			syncData[i] = adminData.SyncFileData{
				SyncFile: s,
			}
		}

		paging := data.CreatePaging(*url, routes.SyncPageParam, page, count, syncFilePerPage)

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
			data.Breadcrumb{
				Label: i18n.T(loc, "nav.page.admin.sync_run_files.short", nil),
				Type:  "sync",
			},
		}

		syncCtx := adminData.SyncFilesPageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Paging = paging
		syncCtx.SyncFiles = syncData
		syncCtx.Breadcrumbs = breadcrumbs
		syncCtx.HasSearch = false
		syncCtx.HasFileButton = true
		syncCtx.HasFilter = true
		syncCtx.Filter = filter
		syncCtx.FilterData = data.CreateDropDown(dbo.AllSyncFileStatus, loc, "data.sync_files.status", i18n)

		if err := r.RenderPage(c.Writer, "admin/syncfiles", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}

/*
All sync run file
*/
func SyncFilesListPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		pageStr := c.DefaultQuery(routes.SyncPageParam, "1")
		search := c.Query(routes.SearchParam)
		filterIn := c.QueryArray(routes.FilterParam)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/sync_file", nil, map[string]any{
			"page":   pageStr,
			"search": search,
			"filter": filterIn,
		})

		filter := make([]string, 0, len(filterIn))
		for _, f := range filterIn {
			if f != "" {
				filter = append(filter, f)
			}
		}

		page, err := tpl.ParsePaging(pageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}

		url := routes.BuildAdminSyncFilesPath()
		url.WithUintQuery(routes.SyncPageParam, page)
		if search != "" {
			url.WithParam(routes.SearchParam, search)
		}
		if len(filter) > 0 {
			url.WithArrayParams(routes.FilterParam, filter)
		}

		database := db.GetDatabase()

		syncs, err := dao.QuerySyncFileBySearchByStatusPaged(database, ctx, search, filter, (page-1)*syncFilePerPage, syncFilePerPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		count, err := dao.CountSyncFileBySearchByStatus(database, ctx, search, filter)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		syncData := make([]adminData.SyncFileData, len(syncs))
		for i, s := range syncs {
			syncData[i] = adminData.SyncFileData{
				SyncFile: s,
			}
		}

		paging := data.CreatePaging(*url, routes.SyncPageParam, page, count, syncFilePerPage)

		breadcrumbs := data.Breadcrumbs{
			tpl.GetAdminMain(loc, i18n),
			data.Breadcrumb{
				Label: i18n.T(loc, "nav.page.admin.sync_files.short", nil),
				Type:  "sync",
			},
		}

		syncCtx := adminData.SyncFilesPageContext{}
		pageCtx := syncCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "sync", data.SurfaceAdmin)
		syncCtx.Paging = paging
		syncCtx.SyncFiles = syncData
		syncCtx.Breadcrumbs = breadcrumbs
		syncCtx.HasSearch = true
		syncCtx.SearchField = search
		syncCtx.HasFileButton = true
		syncCtx.HasFilter = true
		syncCtx.Filter = filter
		syncCtx.FilterData = data.CreateDropDown(dbo.AllSyncFileStatus, tpl.L(c), "data.sync_files.status", i18n)

		if err := r.RenderPage(c.Writer, "admin/syncfiles", syncCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
