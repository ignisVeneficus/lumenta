package public

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
	"github.com/ignisVeneficus/lumenta/utils"
)

func TagsImagePrevNext(database *sql.DB, c context.Context, dboAcl dbo.ACLContext, tagId dbo.TagID, image dbo.Image) (*dbo.ImageTitle, *dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "server/page/public/tag/image/prev_next", image.ID, map[string]any{
		"tag_id":   tagId,
		"image_id": image.ID,
	})
	var prev *dbo.ImageTitle = nil
	var next *dbo.ImageTitle = nil

	n, err := dao.QueryImageIDByTagACLNext(database, ctx, tagId, *image.ID, image.TakenAt, image.Filename, dboAcl, 0, 1)
	if err != nil {
		return prev, next, err
	}
	if len(n) > 0 {
		next = &(n[0])
	}
	p, err := dao.QueryImageIDByTagACLPrev(database, ctx, tagId, *image.ID, image.TakenAt, image.Filename, dboAcl, 0, 1)
	if err != nil {
		return prev, next, err
	}
	if len(p) > 0 {
		prev = &(p[0])
	}
	logging.Exit(logScope, "ok", nil)
	return prev, next, nil
}

func TagImagePage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		tagIdStr := c.Param("tid")
		imageIdStr := c.Param("iid")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "0")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/public/tag/image", imageIdStr, map[string]any{
			"tag_id":     tagIdStr,
			"image_id":   imageIdStr,
			"image_page": iPageStr,
		})
		tagID, err := tpl.ParseTagID(tagIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid tag id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
			return
		}

		imageID, err := tpl.ParseImageID(imageIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		dbTagID := dbo.TagID(tagID)
		dbImageID := dbo.ImageID(imageID)

		thisTag, err := dao.GetTagByIDACL(database, ctx, dbTagID, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "tag", routes.CreateTagsRootPath(), uint64(tagID))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		image, err := dao.GetImageByIdACLWTags(database, ctx, dbImageID, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "image", routes.CreateTagsRootPath(), uint64(imageID))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		generator := func(img routes.ImageID) *string {
			return utils.PtrString(routes.CreateTagImagePath(tagID, img))
		}

		queryNext := func(c context.Context, image dbo.Image, start int, qty int) ([]dbo.ImageTitle, error) {
			return dao.QueryImageIDByTagACLNext(database, ctx, dbTagID, *image.ID, image.TakenAt, image.Filename, acl.ACLContext, uint64(start), uint64(qty))
		}
		queryPrev := func(c context.Context, image dbo.Image, start int, qty int) ([]dbo.ImageTitle, error) {
			return dao.QueryImageIDByTagACLPrev(database, ctx, dbTagID, *image.ID, image.TakenAt, image.Filename, acl.ACLContext, uint64(start), uint64(qty))
		}
		pagingUrlGenerator := func(imageId routes.ImageID, page int) string {
			return routes.BuildTagImagePath(tagID, imageId).WithImageIntPaging(page).String()
		}

		thumbnails, err := data.GenerateThumbnail(ctx, queryPrev, queryNext, image, iPage, generator, pagingUrlGenerator)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		prev, next, err := TagsImagePrevNext(database, ctx, acl.ACLContext, dbTagID, image)

		breadcrumbs, err := tpl.BuildTagBreadcumb(database, ctx, thisTag, false)
		title := image.GetTitle()
		breadcrumbs = append(breadcrumbs, data.Breadcrumb{
			Link: tplData.Link{
				Label: title,
			},
			Type: "image",
		})

		var nextThumb *data.Thumbnail
		var prevThumb *data.Thumbnail

		if next != nil {
			nextThumb = data.CreateThumbnail(*next, generator)
		}
		if prev != nil {
			prevThumb = data.CreateThumbnail(*prev, generator)
		}
		tplImage, err := tpl.CreateImage(ctx, cfg, database, image, acl.ACLContext)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		tagImagePageCtx := data.ImagePageContext{
			Image:      tplImage,
			Thumbnails: &thumbnails,
			Next:       nextThumb,
			Prev:       prevThumb,
			Up:         routes.CreateTagPath(tagID),
		}
		tagImagePageCtx.Breadcrumbs = breadcrumbs

		pageCtx := tagImagePageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)

		if err := r.RenderPage(c.Writer, "public/image", tagImagePageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
