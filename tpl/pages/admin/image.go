package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func resolveFocus(img dbo.Image) data.Focus {
	return data.ResolveFocus(img.FocusX, img.FocusY, data.ImageFocusMode(img.FocusMode))
}

func createHistoryDate(database *sql.DB, ctx context.Context, root, path, filename, ext string) (adminData.ImageSync, error) {
	syncData := adminData.ImageSync{}
	lastSyncs, err := dao.QuerySyncFileByFilePathPaged(database, ctx, root, path, filename, ext, 0, 1)
	if err != nil {
		return syncData, err
	}
	var lastId dbo.SyncFileID = 0
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

func ImagePage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		imageIDStr := c.Param("id")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/image", imageIDStr, map[string]any{
			"image": imageIDStr,
		})

		imageID, err := tpl.ParseImageID(imageIDStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image Id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image Id"})
			return
		}

		database := db.GetDatabase()
		dbImageID := dbo.ImageID(imageID)
		// Image
		image, err := dao.GetImageByIdWTags(database, ctx, dbImageID)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfaceAdmin, "image", routes.CreateAdminRootPath(), uint64(imageID))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// Last sync / history
		syncData, err := createHistoryDate(database, ctx, image.Root, image.Path, image.Filename, image.Ext)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Album related query
		albums, err := dao.QueryAlbumsIdByCover(database, ctx, dbImageID)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// form for change
		form := adminData.ImageForm{
			ACLLevel:  strconv.FormatUint(uint64(image.ACLLevel), 10),
			ACLUserID: strconv.FormatUint(uint64(image.ACLUserID), 10),
		}
		// tags
		forest := tplData.ForestFromTags(image.Tags, func(id uint64) string {
			return ""
		})
		tplData.SetTagsMeaning(forest, cfg.Presentation.TagMeaningConfig)

		imageCtx := adminData.ImagePageContext{}
		pageCtx := imageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "image", tplData.SurfaceAdmin)
		filePath := image.Path + "/" + image.Filename + "." + image.Ext
		imageCtx.Breadcrumbs = createFsBreadcrumbs(image.Root, filePath)
		imageCtx.Image = adminData.PageImage{
			Image:         image,
			Realpath:      tpl.CreateSpacePath(image.PathFull()),
			ComputedFocus: resolveFocus(image),
			Sync:          syncData,
			Aspect:        grid.ClassifyAspect(int(image.Width), int(image.Height)),
			Form:          form,
			Tags:          *forest,
		}
		if image.Latitude != nil && image.Longitude != nil {
			imageCtx.Image.SingleMap = &tplData.SingleMap{
				Lat:  *image.Latitude,
				Long: *image.Longitude,
			}
		}
		albumIDs := make([]routes.AlbumID, 0)
		for _, ai := range albums {
			albumIDs = append(albumIDs, routes.AlbumID(ai))
		}
		imageCtx.Image.Covers = albumIDs

		if err := r.RenderPage(c.Writer, "admin/image", imageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
