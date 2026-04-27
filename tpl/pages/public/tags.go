package public

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
)

var (
	tagsPerPage  uint64 = 6 // one row
	imagePerPage uint64 = 24
)

func BuildFolders(ctx context.Context, database *sql.DB, tagId uint64, acl dbo.ACLContext, page uint64, url data.URLBuilder) (data.Folders, error) {
	type tagImage struct {
		dbo.TagWCount
		ImageId *uint64
	}

	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"tag":  tagId,
		"page": page,
	})
	qty, err := dao.CountTagsByParentACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.Folders{}, err
	}
	tags, err := dao.QueryTagsByParentACLPaged(database, ctx, tagId, acl, (page-1)*tagsPerPage, tagsPerPage)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.Folders{}, err
	}
	paging := data.CreatePaging(url, data.FolderPageParam, page, qty, tagsPerPage)

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
			logging.ExitErr(logg, err)
			return data.Folders{}, err
		}
		tagsItems = append(tagsItems, dir)
	}

	mapping := func(tag tagImage) data.Folder {
		return data.Folder{
			URL:   template.URL(routes.CreateTagPath(*tag.ID)),
			Name:  tag.Name,
			Info:  fmt.Sprintf("%d images", tag.Count),
			Image: tag.ImageId,
		}
	}

	folders := data.CreateFolders(tagsItems, paging, mapping)
	logging.Exit(logg, "ok", nil)
	return folders, nil
}
func BuildImageGrid(ctx context.Context, database *sql.DB, tagId uint64, acl dbo.ACLContext, images []dbo.Image, page uint64, url data.URLBuilder, cfg presentation.PresentationConfig) (data.ImageGrid, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"page": page,
	})
	qty, err := dao.CountImageByTagACL(database, ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.ImageGrid{}, err
	}
	paging := data.CreatePaging(url, data.ImagePageParam, page, qty, imagePerPage)
	makeURL := func(id uint64) string {
		return routes.CreateTagImagePath(tagId, id)
	}
	return data.ImageGrid{
		Paging: &paging,
		Images: grid.BuildGrid(images, cfg.Grid, tagId*page, makeURL),
	}, nil
}

func TagPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		tagIdStr := c.Param("tid")
		tPageStr := c.DefaultQuery(data.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(data.ImagePageParam, "1")
		logg := logging.Enter(c, "page.public.tags", map[string]any{
			"tag_id":      tagIdStr,
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

		tagId, err := tpl.ParseID(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
			return
		}

		tPage, err := tpl.ParsePaging(tPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid subTag_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid directory page"})
			return
		}

		iPage, err := tpl.ParsePaging(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		url := routes.BuildTagPath(tagId)
		url.WithFolderPaging(tPage).WithImagePaging(iPage)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		thisTag, err := dao.GetTagByIDACL(database, c, tagId, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			pages.Soft404(r, cfg, c, data.SurfacePublic, "tag", routes.CreateTagsRootPath(), tagId)
			return
		case err != nil:
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		folders, err := BuildFolders(c, database, tagId, acl.ACLContext, tPage, *url)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		images, err := dao.QueryImageByTagACLPaged(database, c, tagId, acl.ACLContext, (iPage-1)*imagePerPage, imagePerPage)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		imageGrid, err := BuildImageGrid(c, database, tagId, acl.ACLContext, images, iPage, *url, cfg.Presentation)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		breadcrumbs, err := tpl.BuildTagBreadcumb(database, c, thisTag, true)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		tagForrest := dbo.TagsTree{}
		for _, img := range images {
			tagForrest = append(tagForrest, img.Tags...)
		}
		tagForrest = dbo.MergeTagsTrees(tagForrest)

		apiUrl := routes.BuildApiTagPath(tagId).WithImagePaging(iPage)
		multiMap := data.MultiMap{
			APIURL:      apiUrl.String(),
			Cluster:     true,
			Popup:       true,
			Hover:       true,
			NrMaxPoints: 100,
		}

		tagPageCtx := data.FolderPageContext{}
		pageCtx := tagPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)
		tagPageCtx.Breadcrumbs = breadcrumbs
		tagPageCtx.PageCards = folders
		tagPageCtx.ImageGrid = imageGrid
		tagPageCtx.ImageTags = data.ImageTags(tagForrest)
		tagPageCtx.Map = multiMap

		logging.Exit(logg, "ok", nil)
		if err := r.RenderPage(c.Writer, "public/folders", tagPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logg, err)
			return
		}
		logging.Exit(logg, "ok", nil)

	}
}
