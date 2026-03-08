package public

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
	"github.com/ignisVeneficus/lumenta/utils"
)

func TagsImagePrevNext(database *sql.DB, c *gin.Context, dboAcl dao.ACLContext, tagId uint64, image dbo.Image) (*dbo.ImageTitle, *dbo.ImageTitle, error) {
	logg := logging.Enter(c, "page.public.tags.prevNext", nil)
	var prev *dbo.ImageTitle = nil
	var next *dbo.ImageTitle = nil

	n, err := dao.QueryImageIDByTagACLNext(database, c, tagId, *image.ID, image.TakenAt, image.Filename, dboAcl, 0, 1)
	if err != nil {
		return prev, next, err
	}
	if len(n) > 0 {
		next = &(n[0])
	}
	p, err := dao.QueryImageIDByTagACLPrev(database, c, tagId, *image.ID, image.TakenAt, image.Filename, dboAcl, 0, 1)
	if err != nil {
		return prev, next, err
	}
	if len(p) > 0 {
		prev = &(p[0])
	}
	logging.Exit(logg, "ok", nil)
	return prev, next, nil

}

func TagImagePage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagIdStr := c.Param("tid")
		imageIdStr := c.Param("iid")
		iPageStr := c.DefaultQuery(imagePageName, "0")
		logg := logging.Enter(c, "page.public.tags.image", map[string]any{
			"tag_id":     tagIdStr,
			"image_id":   imageIdStr,
			"image_page": iPageStr,
		})
		tagId, err := strconv.Atoi(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
			return
		}

		imageId, err := strconv.Atoi(imageIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid subTag_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		dboAcl := tpl.CreateDBOACL(acl)

		thisTag, err := dao.GetTagByIDACL(database, c, uint64(tagId), dboAcl)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "tag", routes.CreateTagsRootPath(), uint64(tagId))
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		image, err := dao.GetImageByIdACLWTags(database, c, uint64(imageId), dboAcl)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "image", routes.CreateTagsRootPath(), uint64(imageId))
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		generator := func(img uint64) *string {
			return utils.PtrString(routes.CreateTagImagePath(uint64(tagId), img))
		}

		queryNext := func(c context.Context, image dbo.Image, start int, qty int) ([]dbo.ImageTitle, error) {
			return dao.QueryImageIDByTagACLNext(database, c, uint64(tagId), *image.ID, image.TakenAt, image.Filename, dboAcl, uint64(start), uint64(qty))
		}
		queryPrev := func(c context.Context, image dbo.Image, start int, qty int) ([]dbo.ImageTitle, error) {
			return dao.QueryImageIDByTagACLPrev(database, c, uint64(tagId), *image.ID, image.TakenAt, image.Filename, dboAcl, uint64(start), uint64(qty))
		}
		pagingUrlGenerator := func(imageId uint64, page int) string {
			return routes.BuildTagImagePath(uint64(tagId), imageId).WithIntQuery(imagePageName, page).String()
		}

		thumbnails, err := data.GenerateThumbnail(c, queryPrev, queryNext, image, iPage, generator, pagingUrlGenerator)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		prev, next, err := TagsImagePrevNext(database, c, dboAcl, uint64(tagId), image)

		url := routes.BuildTagImagePath(uint64(tagId), uint64(imageId))
		url.WithIntQuery(imagePageName, iPage)

		breadcrumbs, err := tpl.BuildTagBreadcumb(database, c, thisTag, false)
		title := image.GetTitle()
		breadcrumbs = append(breadcrumbs, data.Breadcrumb{
			Label: title,
			Type:  "image",
		})

		var nextThumb *data.Thumbnail
		var prevThumb *data.Thumbnail

		if next != nil {
			nextThumb = data.CreateThumbnail(*next, generator)
		}
		if prev != nil {
			prevThumb = data.CreateThumbnail(*prev, generator)
		}

		tagImagePageCtx := data.ImagePageContext{
			Breadcrumbs: breadcrumbs,
			Image:       tpl.CreateImage(c, cfg, image),
			Thumbnails:  thumbnails,
			Next:        nextThumb,
			Prev:        prevThumb,
		}
		pageCtx := tagImagePageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)

		if err := r.RenderPage(c.Writer, "public/image", tagImagePageCtx); err != nil {
			c.String(500, err.Error())
		}

	}
}
