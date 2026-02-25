package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	focusData "github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func createSpacePath(path, filename, ext string) string {
	ret := path + "/" + filename + "." + ext
	return strings.ReplaceAll(ret, "/", "/\u200B")
}

func resolveFocus(img dbo.Image) focusData.Focus {
	return focusData.ResolveFocus(img.FocusX, img.FocusY, focusData.ImageFocusMode(img.FocusMode))
}

func ImagePage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		imageIDStr := c.Param("id")
		logg := logging.Enter(c, "admin.ImagePage", map[string]any{
			"image": imageIDStr,
		})

		imageID, err := strconv.Atoi(imageIDStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image Id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image Id"})
			return
		}

		database := db.GetDatabase()
		image, err := dao.GetImageByIdWTags(database, c, uint64(imageID))
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfaceAdmin, "image", routes.CreateAdminRootPath(), uint64(imageID))
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		lastSync := dbo.SyncRun{}
		if image.LastSeenSync != nil {
			lastSync, err = dao.GetSyncRunByID(database, c, *image.LastSeenSync)
			if err != nil {
				logging.ErrorContinue(logg, err, map[string]any{"sync_id": image.LastSeenSync})
			}
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)

		imageCtx := adminData.ImageContext{}
		pageCtx := imageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "image", data.SurfaceAdmin)
		filePath := image.Path + "/" + image.Filename + "." + image.Ext
		imageCtx.Breadcrumbs = createFsBreadcrumbs(image.Root, filePath)
		imageCtx.Image = adminData.PageImage{
			Image:         image,
			Realpath:      createSpacePath(image.Path, image.Filename, image.Ext),
			ComputedFocus: resolveFocus(image),
			Sync:          lastSync,
			Aspect:        grid.ClassifyAspect(int(image.Width), int(image.Height)),
		}
		// TODO: get users

		if err := r.RenderPage(c.Writer, "admin/image", imageCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
