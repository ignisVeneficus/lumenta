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
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	albumsPerPage     uint64 = 6 // two row
	imagePerAlbumPage uint64 = 24
)

func BuildAlbumFolders(c context.Context, database *sql.DB, loc string, i18n *i18n.Service, albumId *dbo.AlbumID, acl dbo.ACLContext, page uint64, url routes.URLBuilder) (tplData.Folders, error) {
	type albumImage struct {
		dbo.Album
		ImageId    *routes.ImageID
		AlbumCount int
		ImageCount int
	}

	logScope, ctx := logging.Enter(c, "server/page/public/album/buildDirGrid", albumId, map[string]any{
		"album_id": albumId,
		"page":     page,
	})
	qty, err := dao.CountAlbumByParentACL(database, ctx, albumId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.Folders{}, err
	}
	albums, err := dao.QueryAlbumByParentACLPaged(database, ctx, albumId, acl, (page-1)*albumsPerPage, albumsPerPage)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.Folders{}, err
	}
	paging := tplData.CreatePaging(url, routes.FolderPageParam, page, qty, albumsPerPage)

	albumsItems := []albumImage{}
	for _, a := range albums {
		dir := albumImage{
			Album:      a,
			AlbumCount: 0,
			ImageCount: 0,
		}
		if a.CoverImageID != nil {
			img, err := dao.GetImageByIdACL(database, ctx, *a.CoverImageID, acl)
			switch {
			case err == nil:
				dir.ImageId = (*routes.ImageID)(img.ID)
			case !errors.Is(err, dao.ErrDataNotFound):
				logging.ExitErr(logScope, err)
				return tplData.Folders{}, err
			}
		}
		imagesCount, err := dao.CountImageByAlbumDescendantIDsByACL(database, ctx, *a.ID, acl)
		if err != nil {
			logging.ExitErr(logScope, err)
			return tplData.Folders{}, err
		}
		dir.ImageCount = int(imagesCount)
		albumCount, err := dao.CountAlbumDescendantIDsByACL(database, ctx, *a.ID, acl)
		if err != nil {
			logging.ExitErr(logScope, err)
			return tplData.Folders{}, err
		}
		dir.AlbumCount = int(albumCount)

		dao.CountAlbumDescendantIDsByACL(database, ctx, *a.ID, acl)
		albumsItems = append(albumsItems, dir)
	}

	mapping := func(album albumImage) tplData.Folder {
		return tplData.Folder{
			URL:         template.URL(routes.CreateAlbumPath(routes.AlbumID(*album.ID))),
			Name:        album.Name,
			Description: utils.FromStringPtr(album.Description),
			Info: i18n.T(loc, "page.public.album.nr_content", map[string]any{
				"images": album.ImageCount,
				"albums": album.AlbumCount,
			}),
			Image: album.ImageId,
		}
	}

	folders := tplData.CreateFolders(albumsItems, paging, mapping)
	logging.Exit(logScope, "ok", nil)
	return folders, nil
}
func BuildAlbumImageGrid(c context.Context, database *sql.DB, albumId dbo.AlbumID, acl dbo.ACLContext, images []dbo.Image, page uint64, url routes.URLBuilder, cfg presentation.PresentationConfig) (tplData.ImageGrid, error) {
	logScope, ctx := logging.Enter(c, "server/page/public/album/buildDirGrid", albumId, map[string]any{
		"album_id": albumId,
		"page":     page,
	})
	qty, err := dao.CountImageByAlbumACL(database, ctx, albumId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.ImageGrid{}, err
	}
	paging := tplData.CreatePaging(url, routes.ImagePageParam, page, qty, imagePerAlbumPage)
	makeURL := func(id routes.ImageID) string {
		return routes.CreateAlbumImagePath(routes.AlbumID(albumId), id)
	}
	logging.Exit(logScope, "ok", nil)
	return tplData.ImageGrid{
		Paging: &paging,
		Images: grid.BuildGrid(images, cfg.Grid, uint64(albumId)*page, makeURL),
	}, nil
}

func AlbumPage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		albumIdStr := c.Param("aid")
		tPageStr := c.DefaultQuery(routes.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/public/album", albumIdStr, map[string]any{
			"album_id":    albumIdStr,
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

		albumID, err := tpl.ParseAlbumID(albumIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid album id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid album id"})
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

		url := routes.BuildAlbumPath(albumID)
		url.WithFolderPaging(tPage).WithImagePaging(iPage)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		dbAlbumId := dbo.AlbumID(albumID)
		thisAlbum, err := dao.GetAlbumByIDACL(database, ctx, dbAlbumId, acl.ACLContext)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "album", routes.CreateTagsRootPath(), uint64(albumID))
			return
		case err != nil:
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		folders, err := BuildAlbumFolders(ctx, database, loc, i18n, &dbAlbumId, acl.ACLContext, tPage, *url)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		images, err := dao.QueryImageByAlbumACLPaged(database, ctx, dbAlbumId, acl.ACLContext, (iPage-1)*imagePerTagPage, imagePerTagPage)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		imageGrid, err := BuildAlbumImageGrid(ctx, database, dbAlbumId, acl.ACLContext, images, iPage, *url, cfg.Presentation)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		breadcrumbs, err := tpl.BuildAlbumBreadcumb(database, ctx, thisAlbum, acl.ACLContext, true)
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

		apiUrl := routes.BuildApiAlbumCoordPath(albumID).WithImagePaging(iPage)
		multiMap := tplData.MultiMap{
			APIURL:      apiUrl.String(),
			Cluster:     true,
			Popup:       true,
			Hover:       true,
			NrMaxPoints: 100,
		}

		albumPageCtx := tplData.AlbumPageContext{}
		pageCtx := albumPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "album", tplData.SurfacePublic)
		albumPageCtx.Breadcrumbs = breadcrumbs
		albumPageCtx.PageCards = folders
		albumPageCtx.ImageGrid = imageGrid
		albumPageCtx.ImageTags = *forest
		albumPageCtx.Map = multiMap
		albumPageCtx.AlbumDescription = utils.FromStringPtr(thisAlbum.Description)
		albumPageCtx.AlbumID = albumID

		logging.Exit(logScope, "ok", nil)
		if err := r.RenderPage(c.Writer, "public/albums", albumPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
func AlbumsRootPage(r *tpl.TemplateResolver, cfg config.Config, i18n *i18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := tpl.L(c)
		tPageStr := c.DefaultQuery(routes.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/public/album/all", nil, map[string]any{
			"subTag_page": tPageStr,
			"image_page":  iPageStr,
		})

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

		url := routes.BuildAlbumsRootPath()
		url.WithFolderPaging(tPage).WithImagePaging(iPage)

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		folders, err := BuildAlbumFolders(ctx, database, loc, i18n, nil, acl.ACLContext, tPage, *url)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		breadcrumbs := tplData.Breadcrumbs{
			tplData.Breadcrumb{
				Link: tplData.Link{
					LabelKey: "nav.page.public.albums.short",
				},
				Type: "albums",
			},
		}

		apiUrl := routes.BuildApiAlbumsRootCoordPath().WithImagePaging(iPage)
		multiMap := tplData.MultiMap{
			APIURL:      apiUrl.String(),
			Cluster:     true,
			Popup:       true,
			Hover:       true,
			NrMaxPoints: 100,
		}

		albumPageCtx := tplData.AlbumPageContext{}
		pageCtx := albumPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "album", tplData.SurfacePublic)
		albumPageCtx.Breadcrumbs = breadcrumbs
		albumPageCtx.PageCards = folders
		albumPageCtx.ImageGrid = tplData.ImageGrid{}
		albumPageCtx.ImageTags = *tplData.NewFlatForest().Build()
		albumPageCtx.Map = multiMap

		logging.Exit(logScope, "ok", nil)
		if err := r.RenderPage(c.Writer, "public/albums", albumPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}
