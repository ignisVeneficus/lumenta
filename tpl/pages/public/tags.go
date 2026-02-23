package public

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
)

var (
	tagsPerPage   = 6 // one row
	imagePerPage  = 24
	tagPageName   = "tPage"
	imagePageName = "iPage"
)

func createTagsBreadcrumbs(tags []dbo.Tag) data.Breadcrumbs {
	res := data.Breadcrumbs{
		data.Breadcrumb{
			Label: "Tags",
			Link:  template.URL(routes.CreateTagsRootPath()),
			Type:  "page",
			Title: "View all tags",
		},
	}
	for i := 0; i < len(tags)-1; i++ {
		brc := data.Breadcrumb{
			Label: tags[i].Name,
			Link:  template.URL(routes.CreateTagPath(*tags[i].ID)),
			Type:  "tag",
			Title: fmt.Sprintf("View: %s", tags[i].Name),
		}
		res = append(res, brc)
	}
	brc := data.Breadcrumb{
		Label: tags[len(tags)-1].Name,
		Type:  "tag",
	}
	res = append(res, brc)

	return res

}

func BuildTagGrid(ctx context.Context, database *sql.DB, tagId uint64, acl dao.ACLContext, page int, url data.URLBuilder) (data.PageTags, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"tag":  tagId,
		"page": page,
	})
	qty, err := dao.CountTagsByParentACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.PageTags{}, err
	}
	tags, err := dao.QueryTagsByParentACLPaged(database, ctx, tagId, acl, uint64((page-1)*tagsPerPage), uint64(tagsPerPage))
	if err != nil {
		logging.ExitErr(logg, err)
		return data.PageTags{}, err
	}
	paging := tpl.CreatePaging(url, tagPageName, page, qty, tagsPerPage)

	tagsItems := []data.PageTag{}
	for _, t := range tags {
		dir := data.PageTag{
			Name:   t.Name,
			ImgQty: t.Count,
			URL:    template.URL(routes.CreateTagPath(*t.ID)),
		}
		img, err := dao.GetImageIdByTagACLFirstHash(database, ctx, *t.ID, acl)
		switch {
		case err == nil:
			dir.Image = img
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			return data.PageTags{}, err
		}
		tagsItems = append(tagsItems, dir)
	}
	ret := data.PageTags{
		Tags:   tagsItems,
		Paging: paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil
}
func BuildImageGrid(ctx context.Context, database *sql.DB, tagId uint64, acl dao.ACLContext, images []dbo.Image, page int, url data.URLBuilder, cfg presentation.PresentationConfig) (data.ImageGrid, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"page": page,
	})
	qty, err := dao.CountImageByTagACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.ImageGrid{}, err
	}
	paging := tpl.CreatePaging(url, imagePageName, page, qty, imagePerPage)
	makeURL := func(id uint64) string {
		return routes.CreateTagImagePath(tagId, id)
	}
	return data.ImageGrid{
		Paging: paging,
		Images: grid.BuildGrid(images, cfg.Grid, tagId*uint64(page), makeURL),
	}, nil
}

func TagPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagIdStr := c.Param("id")
		tPageStr := c.DefaultQuery(tagPageName, "1")
		iPageStr := c.DefaultQuery(imagePageName, "1")
		logg := logging.Enter(c, "public.tagPage", map[string]any{
			"tag_id":      tagIdStr,
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

		tagId, err := strconv.Atoi(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
			return
		}

		tPage, err := strconv.Atoi(tPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid subTag_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid directory page"})
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		url := routes.BuildTagPath(uint64(tagId))
		url.WithIntQuery(tagPageName, tPage).WithIntQuery(imagePageName, iPage)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		thisTag, err := dao.GetTagByIDACL(database, c, uint64(tagId), tpl.CreateDBOACL(acl))
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusNotFound)
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		path := []dbo.Tag{thisTag}
		for thisTag.ParentID != nil {
			thisTag, err = dao.GetTagByID(database, c, *thisTag.ParentID)
			if err != nil {
				logging.ExitErr(logg, err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			path = append([]dbo.Tag{thisTag}, path...)
		}

		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		dboAcl := tpl.CreateDBOACL(acl)
		tags, err := BuildTagGrid(c, database, uint64(tagId), dboAcl, tPage, *url)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		images, err := dao.QueryImageByTagACLPaged(database, c, uint64(tagId), dboAcl, uint64((iPage-1)*imagePerPage), uint64(imagePerPage))
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		imageGrid, err := BuildImageGrid(c, database, uint64(tagId), dboAcl, images, iPage, *url, cfg.Presentation)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		tagForrest := dbo.TagsTree{}
		for _, img := range images {
			tagForrest = append(tagForrest, img.Tags...)
		}
		tagForrest = dbo.MergeTagsTrees(tagForrest)

		tagPageCtx := data.TagPageContext{}
		pageCtx := tagPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)
		tagPageCtx.Breadcrumbs = createTagsBreadcrumbs(path)
		tagPageCtx.PageTags = tags
		tagPageCtx.ImageGrid = imageGrid
		tagPageCtx.ImageTags = data.ImageTags(tagForrest)

		if err := r.RenderPage(c.Writer, "public/tags", tagPageCtx); err != nil {
			c.String(500, err.Error())
		}

	}
}
