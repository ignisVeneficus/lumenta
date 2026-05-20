package public

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func ImagePage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		imageIdStr := c.Param("id")
		logScope, ctx := logging.Enter(c, "server/page/public/image", imageIdStr, map[string]any{
			"image_id": imageIdStr,
		})

		imageId, err := tpl.ParseImageID(imageIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		image, err := dao.GetImageByIdACLWTags(database, ctx, dbo.ImageID(imageId), acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "image", routes.CreateTagsRootPath(), uint64(imageId))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		imagePageCtx := data.ImagePageContext{
			Image:      tpl.CreateImage(ctx, cfg, image),
			Thumbnails: nil,
			Next:       nil,
			Prev:       nil,
			Up:         "/",
		}
		imagePageCtx.Breadcrumbs = data.Breadcrumbs{
			data.Breadcrumb{
				Label: image.GetTitle(),
				Type:  "image",
			},
		}

		pageCtx := imagePageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "images", data.SurfacePublic)

		if err := r.RenderPage(c.Writer, "public/image", imagePageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
