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
	tagsPerPage     uint64 = 6 // one row
	imagePerTagPage uint64 = 24
)

func BuildTagFolders(c context.Context, database *sql.DB, loc string, i18n *i18n.Service, tagId dbo.TagID, acl dbo.ACLContext, page uint64, url routes.URLBuilder) (tplData.Folders, error) {
	type tagImage struct {
		dbo.TagWCount
		ImageId *routes.ImageID
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
	paging := tplData.CreatePaging(url, routes.FolderPageParam, page, qty, tagsPerPage)

	tagsItems := []tagImage{}
	for _, t := range tags {
		dir := tagImage{
			TagWCount: t,
		}
		img, err := dao.GetImageIdByTagACLFirstHash(database, ctx, *t.ID, acl)
		switch {
		case err == nil:
			dir.ImageId = (*routes.ImageID)(&img)
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			return tplData.Folders{}, err
		}
		tagsItems = append(tagsItems, dir)
	}

	mapping := func(folder tagImage) tplData.Folder {
		return tplData.Folder{
			URL:   template.URL(routes.CreateTagPath(routes.TagID(*folder.ID))),
			Name:  folder.Name,
			Info:  i18n.T(loc, "page.public.tag.nr_images", map[string]any{"images": folder.Count}),
			Image: folder.ImageId,
		}
	}

	folders := tplData.CreateFolders(tagsItems, paging, mapping)
	logging.Exit(logScope, "ok", nil)
	return folders, nil
}
func BuildTagImageGrid(c context.Context, database *sql.DB, tagId dbo.TagID, acl dbo.ACLContext, images []dbo.Image, page uint64, url routes.URLBuilder, cfg presentation.PresentationConfig) (tplData.ImageGrid, error) {
	logScope, ctx := logging.Enter(c, "server/page/public/tag/buildDirGrid", tagId, map[string]any{
		"tag_id": tagId,
		"page":   page,
	})
	qty, err := dao.CountImageByTagACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.ImageGrid{}, err
	}
	paging := tplData.CreatePaging(url, routes.ImagePageParam, page, qty, imagePerTagPage)
	makeURL := func(id routes.ImageID) string {
		return routes.CreateTagImagePath(routes.TagID(tagId), id)
	}
	logging.Exit(logScope, "ok", nil)
	return tplData.ImageGrid{
		Paging: &paging,
		Images: grid.BuildGrid(images, cfg.Grid, uint64(tagId)*page, makeURL),
	}, nil
}

func TagPage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		tagIdStr := c.Param("tid")
		tPageStr := c.DefaultQuery(routes.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), tagIdStr, "server/page/public/tag", map[string]any{
			"tag_id":      tagIdStr,
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

		tagId, err := tpl.ParseTagID(tagIdStr)
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
		dbTagID := dbo.TagID(tagId)

		thisTag, err := dao.GetTagByIDACL(database, ctx, dbTagID, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "tag", routes.CreateTagsRootPath(), uint64(tagId))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		folders, err := BuildTagFolders(ctx, database, loc, i18n, dbTagID, acl.ACLContext, tPage, *url)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		images, err := dao.QueryImageByTagACLPaged(database, ctx, dbTagID, acl.ACLContext, (iPage-1)*imagePerTagPage, imagePerTagPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		imageGrid, err := BuildTagImageGrid(ctx, database, dbTagID, acl.ACLContext, images, iPage, *url, cfg.Presentation)
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

		flatForrest := tplData.NewFlatForest()
		mapper := func(t *dbo.Tag) tplData.ViewTreeNode {
			return tplData.ViewTreeNode{
				ID:       uint64(*t.ID),
				ParentID: (*uint64)(t.ParentID),
				Label:    t.Name,
				URL:      template.URL(routes.CreateTagPath(routes.TagID(*t.ID))),
			}
		}
		for _, img := range images {
			tagsList := tplData.MapToViewNodes(img.Tags, mapper)
			flatForrest.Add(tagsList)
		}
		forest := flatForrest.Build()
		tplData.SetTagsMeaning(forest, cfg.Presentation.TagMeaningConfig)

		apiUrl := routes.BuildApiTagCoordPath(tagId).WithImagePaging(iPage)
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
		if err := r.RenderPage(c.Writer, "public/tags", tagPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
