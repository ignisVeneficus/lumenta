package public

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

func ImagePage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		imageIdStr := c.Param("id")
		logg := logging.Enter(c, "page.public.image", map[string]any{
			"image_id": imageIdStr,
		})

		imageId, err := tpl.ParseID(imageIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		image, err := dao.GetImageByIdACLWTags(database, c, imageId, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "image", routes.CreateTagsRootPath(), imageId)
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		imagePageCtx := data.ImagePageContext{
			Image:      tpl.CreateImage(c, cfg, image),
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
			logging.ExitErr(logg, err)
			return
		}
		logging.Exit(logg, "ok", nil)
	}
}
