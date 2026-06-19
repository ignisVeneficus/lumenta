package admin

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/internal/locale"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
)

func createDashboardStat(label, value string) adminData.DashboardStat {
	return adminData.DashboardStat{
		Label: label,
		Value: value,
	}
}
func createDashboardStatUint64(label string, value uint64) adminData.DashboardStat {
	return adminData.DashboardStat{
		Label: label,
		Value: strconv.FormatUint(value, 10),
	}
}

func getImagesCard(database *sql.DB, ctx context.Context) (adminData.DashboardCard, error) {
	imageCount, err := dao.CountImage(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	aclCount, err := dao.CountImageACLLevels(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	stats := make([]adminData.DashboardStat, 4)
	stats[0] = createDashboardStatUint64("page.admin.main.image.total", imageCount)
	public, ok := aclCount[dbo.DBACLLevelPublic]
	if !ok {
		public = 0
	}
	stats[1] = createDashboardStatUint64("page.admin.main.image.public", public)
	users, ok := aclCount[dbo.DBACLLevelAuthenticated]
	if !ok {
		users = 0
	}
	stats[2] = createDashboardStatUint64("page.admin.main.image.user", users)
	admin, ok := aclCount[dbo.DBACLLevelAdmin]
	if !ok {
		admin = 0
	}
	stats[3] = createDashboardStatUint64("page.admin.main.image.admin", admin)
	return adminData.DashboardCard{
		ID:           "images",
		TitleKey:     "nav.page.admin.images.short",
		IconTitleKey: "nav.page.admin.images.label",
		IconKey:      "nav.page.admin.images.large",
		URL:          routes.CreateAdminFsPath(""),
		Stats:        stats,
	}, nil
}

func getAlbumCard(database *sql.DB, ctx context.Context) (adminData.DashboardCard, error) {
	albums, err := dao.CountAlbum(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	wAlbums, wtAlbums, err := dao.CountImageByAlbumsBinding(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	stats := make([]adminData.DashboardStat, 3)
	stats[0] = createDashboardStatUint64("page.admin.main.album.total", albums)
	stats[1] = createDashboardStatUint64("page.admin.main.album.with_albums", wAlbums)
	stats[2] = createDashboardStatUint64("page.admin.main.album.without_albums", wtAlbums)

	return adminData.DashboardCard{
		ID:           "albums",
		TitleKey:     "nav.page.admin.albums.short",
		IconTitleKey: "nav.page.admin.albums.label",
		IconKey:      "nav.page.admin.albums.large",
		URL:          routes.CreateAdminAlbumsPath(),
		Stats:        stats,
	}, nil
}
func getSyncRunCard(loc string, i18n *i18n.Service, syncRun dbo.SyncRun, fileStats map[dbo.SyncFileStatus]uint64) adminData.DashboardCard {

	dur := tpl.CalcDuration(&syncRun.StartedAt, nil)
	stats := make([]adminData.DashboardStat, 5)
	stats[0] = createDashboardStat("data.sync_runs.status."+string(syncRun.Status)+".short", locale.FormatDuration(dur, loc, i18n))

	newCnt, ok := fileStats[dbo.SyncFileStatusCreated]
	if !ok {
		newCnt = 0
	}
	filteredCnt, ok := fileStats[dbo.SyncFileStatusFilteredOut]
	if !ok {
		filteredCnt = 0
	}
	deletedCnt, ok := fileStats[dbo.SyncFileStatusDeleted]
	if !ok {
		deletedCnt = 0
	}
	errorCnt, ok := fileStats[dbo.SyncFileStatusError]
	if !ok {
		errorCnt = 0
	}

	stats[1] = createDashboardStatUint64("data.sync_files.status.created.short", newCnt)
	stats[2] = createDashboardStatUint64("data.sync_files.status.deleted.short", deletedCnt)
	stats[3] = createDashboardStatUint64("data.sync_files.status.filtered_out.short", filteredCnt)
	stats[4] = createDashboardStatUint64("data.sync_files.status.error.short", errorCnt)

	return adminData.DashboardCard{
		ID:           "sync_runs",
		TitleKey:     "nav.page.admin.sync_runs.short",
		IconTitleKey: "nav.page.admin.sync_runs.label",
		IconKey:      "nav.page.admin.sync_runs.large",
		URL:          routes.CreateAdminSyncRunListPath(),
		Stats:        stats,
	}
}
func getSyncFilesCard(fileStats map[dbo.SyncFileStatus]uint64) adminData.DashboardCard {
	stats := make([]adminData.DashboardStat, 2)
	updatedCnt, ok := fileStats[dbo.SyncFileStatusUpdated]
	if !ok {
		updatedCnt = 0
	}
	notChangedCnt, ok := fileStats[dbo.SyncFileStatusNotChanged]
	if !ok {
		notChangedCnt = 0
	}

	stats[0] = createDashboardStatUint64("data.sync_files.status.updated.short", updatedCnt)
	stats[1] = createDashboardStatUint64("data.sync_files.status.not_changed.short", notChangedCnt)

	return adminData.DashboardCard{
		ID:           "sync_files",
		TitleKey:     "nav.page.admin.sync_files.short",
		IconTitleKey: "nav.page.admin.sync_files.label",
		IconKey:      "nav.page.admin.sync_files.large",
		URL:          routes.CreateAdminSyncFilesPath(),
		Stats:        stats,
	}
}

func MainPage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/main", nil, nil)

		database := db.GetDatabase()

		cards := make([]adminData.DashboardCard, 0, 4)
		imgCard, err := getImagesCard(database, ctx)
		if err != nil {
			logScope.ExitErr(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		cards = append(cards, imgCard)

		albumCard, err := getAlbumCard(database, ctx)
		if err != nil {
			logScope.ExitErr(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		cards = append(cards, albumCard)

		lastSync, err := dao.GetSyncRunLast(database, ctx)
		if err != nil {
			logScope.ExitErr(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		syncStat, err := dao.CountSyncFileStatusBySyncID(database, ctx, *lastSync.ID)
		lastSyncCard := getSyncRunCard(loc, i18n, lastSync, syncStat)
		cards = append(cards, lastSyncCard)

		fileSyncCard := getSyncFilesCard(syncStat)
		cards = append(cards, fileSyncCard)

		mainPageCtx := adminData.MainPageContext{
			Cards: cards,
		}
		pageCtx := mainPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "main", data.SurfaceAdmin)

		if err := r.RenderPage(c.Writer, "admin/main", mainPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logScope.ExitErr(err)
			return
		}
		logScope.Exit("ok", nil)
	}
}
