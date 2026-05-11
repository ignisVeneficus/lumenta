package public

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

var (
	tagsPerPage  uint64 = 6 // one row
	imagePerPage uint64 = 24
)

func BuildFolders(c context.Context, database *sql.DB, tagId uint64, acl dbo.ACLContext, page uint64, url tplData.URLBuilder) (tplData.Folders, error) {
	type tagImage struct {
		dbo.TagWCount
		ImageId *uint64
	}

	logScope, ctx := logging.Enter(c, "server/page/public/tag/buildDirGrid", tagId, map[string]any{
		"tag_id": tagId,
		"page":   page,
	})
	qty, err := dao.CountTagsByParentACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.Folders{}, err
	}
	tags, err := dao.QueryTagsByParentACLPaged(database, ctx, tagId, acl, (page-1)*tagsPerPage, tagsPerPage)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.Folders{}, err
	}
	paging := tplData.CreatePaging(url, tplData.FolderPageParam, page, qty, tagsPerPage)

	tagsItems := []tagImage{}
	for _, t := range tags {
		dir := tagImage{
			TagWCount: t,
		}
		img, err := dao.GetImageIdByTagACLFirstHash(database, ctx, *t.ID, acl)
		switch {
		case err == nil:
			dir.ImageId = &img
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			return tplData.Folders{}, err
		}
		tagsItems = append(tagsItems, dir)
	}

	mapping := func(tag tagImage) tplData.Folder {
		return tplData.Folder{
			URL:   template.URL(routes.CreateTagPath(*tag.ID)),
			Name:  tag.Name,
			Info:  fmt.Sprintf("%d images", tag.Count),
			Image: tag.ImageId,
		}
	}

	folders := tplData.CreateFolders(tagsItems, paging, mapping)
	logging.Exit(logScope, "ok", nil)
	return folders, nil
}
func BuildImageGrid(c context.Context, database *sql.DB, tagId uint64, acl dbo.ACLContext, images []dbo.Image, page uint64, url tplData.URLBuilder, cfg presentation.PresentationConfig) (tplData.ImageGrid, error) {
	logScope, ctx := logging.Enter(c, "server/page/public/tag/buildDirGrid", tagId, map[string]any{
		"tag_id": tagId,
		"page":   page,
	})
	qty, err := dao.CountImageByTagACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.ImageGrid{}, err
	}
	paging := tplData.CreatePaging(url, tplData.ImagePageParam, page, qty, imagePerPage)
	makeURL := func(id uint64) string {
		return routes.CreateTagImagePath(tagId, id)
	}
	logging.Exit(logScope, "ok", nil)
	return tplData.ImageGrid{
		Paging: &paging,
		Images: grid.BuildGrid(images, cfg.Grid, tagId*page, makeURL),
	}, nil
}

func TagPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		tagIdStr := c.Param("tid")
		tPageStr := c.DefaultQuery(tplData.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(tplData.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), tagIdStr, "server/page/public/tag", map[string]any{
			"tag_id":      tagIdStr,
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

		tagId, err := tpl.ParseID(tagIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid tag id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
			return
		}

		tPage, err := tpl.ParsePaging(tPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid subTag_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid directory page"})
			return
		}

		iPage, err := tpl.ParsePaging(iPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		url := routes.BuildTagPath(tagId)
		url.WithFolderPaging(tPage).WithImagePaging(iPage)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		thisTag, err := dao.GetTagByIDACL(database, ctx, tagId, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "tag", routes.CreateTagsRootPath(), tagId)
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		folders, err := BuildFolders(ctx, database, tagId, acl.ACLContext, tPage, *url)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		images, err := dao.QueryImageByTagACLPaged(database, ctx, tagId, acl.ACLContext, (iPage-1)*imagePerPage, imagePerPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		imageGrid, err := BuildImageGrid(ctx, database, tagId, acl.ACLContext, images, iPage, *url, cfg.Presentation)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		breadcrumbs, err := tpl.BuildTagBreadcumb(database, ctx, thisTag, true)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		flatForrest := tplData.NewFlatForrest()
		mapper := func(t *dbo.Tag) tplData.ViewTreeNode {
			return tplData.ViewTreeNode{
				ID:       *t.ID,
				ParentID: t.ParentID,
				Label:    t.Name,
				URL:      template.URL(routes.CreateTagPath(*t.ID)),
			}
		}
		for _, img := range images {
			tagsList := tplData.MapToViewNodes(img.Tags, mapper)
			flatForrest.Add(tagsList)
		}
		forest := flatForrest.Build()
		tplData.SetTagsMeaning(forest, cfg.Presentation.TagMeaningConfig)

		apiUrl := routes.BuildApiTagPath(tagId).WithImagePaging(iPage)
		multiMap := tplData.MultiMap{
			APIURL:      apiUrl.String(),
			Cluster:     true,
			Popup:       true,
			Hover:       true,
			NrMaxPoints: 100,
		}

		tagPageCtx := tplData.FolderPageContext{}
		pageCtx := tagPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", tplData.SurfacePublic)
		tagPageCtx.Breadcrumbs = breadcrumbs
		tagPageCtx.PageCards = folders
		tagPageCtx.ImageGrid = imageGrid
		tagPageCtx.ImageTags = *forest
		tagPageCtx.Map = multiMap

		logging.Exit(logScope, "ok", nil)
		if err := r.RenderPage(c.Writer, "public/folders", tagPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
