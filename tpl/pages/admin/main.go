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
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
)

type cardBuilder func(database *sql.DB, ctx context.Context, loc string, i18n *i18n.Service) (adminData.DashboardCard, error)

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

func getImagesCard(database *sql.DB, ctx context.Context, loc string, i18n *i18n.Service) (adminData.DashboardCard, error) {
	imageCount, err := dao.CountImage(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	aclCount, err := dao.CountImageACLLevels(database, ctx)
	if err != nil {
		return adminData.DashboardCard{}, err
	}
	stats := make([]adminData.DashboardStat, 4)
	stats[0] = createDashboardStatUint64("Total", imageCount)
	public, ok := aclCount[dbo.DBACLLevelPublic]
	if !ok {
		public = 0
	}
	stats[1] = createDashboardStatUint64("Public", public)
	users, ok := aclCount[dbo.DBACLLevelAuthenticated]
	if !ok {
		users = 0
	}
	stats[2] = createDashboardStatUint64("Users", users)
	admin, ok := aclCount[dbo.DBACLLevelAdmin]
	if !ok {
		admin = 0
	}
	stats[3] = createDashboardStatUint64("Admin", admin)
	return adminData.DashboardCard{
		ID:        "images",
		Title:     i18n.T(loc, "nav.page.admin.images.short", nil),
		IconTitle: i18n.T(loc, "nav.page.admin.images.label", nil),
		IconKey:   "nav.page.admin.images.large",
		URL:       routes.CreateAdminFsPath(""),
		Stats:     stats,
	}, nil
}

func getAlbumCard(database *sql.DB, ctx context.Context, loc string, i18n *i18n.Service) (adminData.DashboardCard, error) {
	return adminData.DashboardCard{
		ID:        "albums",
		Title:     i18n.T(loc, "nav.page.admin.albums.short", nil),
		IconTitle: i18n.T(loc, "nav.page.admin.albums.label", nil),
		IconKey:   "nav.page.admin.albums.large",
		URL:       routes.CreateAdminAlbumsPath(),
	}, nil
}
func getSyncRunCard(database *sql.DB, ctx context.Context, loc string, i18n *i18n.Service) (adminData.DashboardCard, error) {
	return adminData.DashboardCard{
		ID:        "sync_runs",
		Title:     i18n.T(loc, "nav.page.admin.sync_runs.short", nil),
		IconTitle: i18n.T(loc, "nav.page.admin.sync_runs.label", nil),
		IconKey:   "nav.page.admin.sync_runs.large",
		URL:       routes.CreateAdminSyncRunListPath(),
	}, nil
}
func getSyncFilesCard(database *sql.DB, ctx context.Context, loc string, i18n *i18n.Service) (adminData.DashboardCard, error) {
	return adminData.DashboardCard{
		ID:        "sync_files",
		Title:     i18n.T(loc, "nav.page.admin.sync_files.short", nil),
		IconTitle: i18n.T(loc, "nav.page.admin.sync_files.label", nil),
		IconKey:   "nav.page.admin.sync_files.large",
		URL:       routes.CreateAdminSyncFilesPath(),
	}, nil
}

func MainPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/main", nil, nil)

		database := db.GetDatabase()

		builders := []cardBuilder{
			getImagesCard,
			getAlbumCard,
			getSyncRunCard,
			getSyncFilesCard,
		}

		cards := make([]adminData.DashboardCard, 0, len(builders))

		for _, b := range builders {
			card, err := b(database, ctx, loc, i18n)
			if err != nil {
				logging.ExitErr(logScope, err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			cards = append(cards, card)
		}

		mainPageCtx := adminData.MainPageContext{
			Cards: cards,
		}
		pageCtx := mainPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "main", data.SurfaceAdmin)

		if err := r.RenderPage(c.Writer, "admin/main", mainPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
