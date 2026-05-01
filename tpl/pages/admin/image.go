package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	focusData "github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func resolveFocus(img dbo.Image) focusData.Focus {
	return focusData.ResolveFocus(img.FocusX, img.FocusY, focusData.ImageFocusMode(img.FocusMode))
}

func createHistoryDate(database *sql.DB, ctx context.Context, root, path, filename, ext string) (adminData.ImageSync, error) {
	syncData := adminData.ImageSync{}
	lastSyncs, err := dao.QuerySyncFileByFilePathPaged(database, ctx, root, path, filename, ext, 0, 1)
	if err != nil {
		return syncData, err
	}
	var lastId uint64 = 0
	if len(lastSyncs) == 0 {
		return syncData, err
	}
	lastSync := lastSyncs[0]
	syncData.LastSync = lastSync.CreatedAt
	syncData.LastStatus = string(lastSync.Status)
	lastId = *lastSync.ID

	lastUpdated, err := dao.GetSyncFileByPathByStatusLast(database, ctx, root, path, filename, ext, dbo.SyncFileStatusUpdated)
	if err != nil {
		if !errors.Is(err, dao.ErrDataNotFound) {
			return syncData, err
		}
	} else {
		if (*lastUpdated.ID) != lastId {
			syncData.LastUpdated = &lastUpdated.CreatedAt
		}
	}
	return syncData, nil
}

func ImagePage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		imageIDStr := c.Param("id")
		logg := logging.Enter(c, "page.admin.image", map[string]any{
			"image": imageIDStr,
		})

		imageID, err := tpl.ParseID(imageIDStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image Id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image Id"})
			return
		}

		database := db.GetDatabase()
		// Image
		image, err := dao.GetImageByIdWTags(database, c, imageID)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfaceAdmin, "image", routes.CreateAdminRootPath(), imageID)
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Last sync / history
		syncData, err := createHistoryDate(database, c, image.Root, image.Path, image.Filename, image.Ext)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Album related query
		albums, err := dao.QueryAlbumsIdByCover(database, c, imageID)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// form for change
		form := adminData.ImageForm{
			ACLLevel:  strconv.FormatUint(uint64(image.ACLLevel), 10),
			ACLUserID: strconv.FormatUint(image.ACLUserID, 10),
		}
		// tags
		forrest := data.ForrestFromTags(image.Tags, func(id uint64) string {
			return ""
		})
		if cfg.Presentation.TagMeaningConfig != nil {
			types := make(map[string][]string)
			for k, v := range cfg.Presentation.TagMeaningConfig.MeaningMap {
				types[string(k)] = v
			}
			forrest.SetTags(types)
			forrest.Populate()
		}

		imageCtx := adminData.ImagePageContext{}
		pageCtx := imageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "image", data.SurfaceAdmin)
		filePath := image.Path + "/" + image.Filename + "." + image.Ext
		imageCtx.Breadcrumbs = createFsBreadcrumbs(image.Root, filePath, loc, i18n)
		imageCtx.Image = adminData.PageImage{
			Image:         image,
			Realpath:      tpl.CreateSpacePath(image.PathFull()),
			ComputedFocus: resolveFocus(image),
			Sync:          syncData,
			Aspect:        grid.ClassifyAspect(int(image.Width), int(image.Height)),
			Form:          form,
			Tags:          forrest,
		}
		if image.Latitude != nil && image.Longitude != nil {
			imageCtx.Image.SingleMap = &data.SingleMap{
				Lat:  *image.Latitude,
				Long: *image.Longitude,
			}
		}
		imageCtx.Image.Covers = albums

		if err := r.RenderPage(c.Writer, "admin/image", imageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logg, err)
			return
		}
		logging.Exit(logg, "ok", nil)
	}
}
