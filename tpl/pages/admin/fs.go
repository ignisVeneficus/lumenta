package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	dirPerPage   uint64 = 24
	imagePerPage uint64 = 48
)

func BuildRootDirGrid(c context.Context, database *sql.DB, page uint64, url routes.URLBuilder) (adminData.FsDirs, error) {
	logScope, ctx := logging.Enter(c, "admin.fsPage.buildRootDirGrid", nil, map[string]any{
		"page": page,
	})
	qty, err := dao.CountImageRoots(database, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsDirs{}, err
	}
	dirs, err := dao.QueryImageRoots(database, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsDirs{}, err
	}
	paging := tplData.CreatePaging(url, routes.FolderPageParam, page, qty, dirPerPage)

	dirItems := []adminData.FsDir{}
	for _, d := range dirs {
		dir := adminData.FsDir{
			Name:   path.Base(d),
			ImgQty: 0,
			URL:    template.URL(routes.BuildAdminFsPath(d).String()),
		}
		img, err := dao.GetImageIdByRootFirstHash(database, ctx, d)
		switch {
		case err == nil:
			dir.Image = routes.ImageID(img)
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			return adminData.FsDirs{}, err
		}
		qty, err := dao.CountImageByRoot(database, ctx, d)
		if err != nil {
			logging.ExitErr(logScope, err)
			return adminData.FsDirs{}, err
		}
		dir.ImgQty = qty
		dirItems = append(dirItems, dir)
	}
	ret := adminData.FsDirs{
		Directories: dirItems,
		Paging:      paging,
	}
	logging.Exit(logScope, "ok", nil)
	return ret, nil
}

func BuildDirGrid(c context.Context, database *sql.DB, root, pagePath string, page uint64, url routes.URLBuilder) (adminData.FsDirs, error) {
	logScope, ctx := logging.Enter(c, "admin.fsPage.buildDirGrid", root+"/"+pagePath, map[string]any{
		"root": root,
		"path": pagePath,
		"page": page,
	})
	qty, err := dao.CountImagePathByParentPath(database, ctx, root, pagePath)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsDirs{}, err
	}
	dirs, err := dao.QueryImagePathByParentPathPaged(database, ctx, root, pagePath, (page-1)*dirPerPage, dirPerPage)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsDirs{}, err
	}
	paging := tplData.CreatePaging(url, routes.FolderPageParam, page, qty, dirPerPage)

	dirItems := []adminData.FsDir{}
	for _, d := range dirs {
		dirPath := path.Join(root, d)
		dir := adminData.FsDir{
			Name:   path.Base(dirPath),
			ImgQty: 0,
			URL:    template.URL(routes.BuildAdminFsPath(dirPath).String()),
		}
		img, err := dao.GetImageIdByPathFirstHash(database, ctx, root, d)
		switch {
		case err == nil:
			dir.Image = routes.ImageID(img)
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logScope, err)
			return adminData.FsDirs{}, err
		}
		qty, err := dao.CountImageByParentPath(database, ctx, root, d)
		if err != nil {
			logging.ExitErr(logScope, err)
			return adminData.FsDirs{}, err
		}
		dir.ImgQty = qty
		dirItems = append(dirItems, dir)
	}
	ret := adminData.FsDirs{
		Directories: dirItems,
		Paging:      paging,
	}
	logging.Exit(logScope, "ok", nil)
	return ret, nil
}
func BuildImageGrid(c context.Context, database *sql.DB, root, path string, page uint64, url routes.URLBuilder) (adminData.FsImages, error) {
	logScope, ctx := logging.Enter(c, "admin.fsPage.buildImageGrid", root+"/"+path, map[string]any{
		"root": root,
		"path": path,
		"page": page,
	})
	if root == "" {
		logging.Exit(logScope, "no root, no image", nil)
		return adminData.FsImages{}, nil
	}
	qty, err := dao.CountImagesByPath(database, ctx, root, path)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsImages{}, err
	}
	images, err := dao.QueryImageWLastSyncWUserByPathPaged(database, ctx, root, path, (page-1)*imagePerPage, imagePerPage)
	if err != nil {
		logging.ExitErr(logScope, err)
		return adminData.FsImages{}, err
	}

	paging := tplData.CreatePaging(url, routes.ImagePageParam, page, qty, imagePerPage)

	imageItems := []adminData.FsImage{}
	for _, i := range images {
		acl := ""
		switch i.ACLLevel {
		case dbo.DBACLLevelPublic:
			acl = "Public"
		case dbo.DBACLLevelAuthenticated:
			if i.User != nil {
				acl = "User: " + utils.FromStringPtr(i.User)
			} else {
				acl = "Users"
			}
		case dbo.DBACLLevelAdmin:
			acl = "Admin"
		}
		img := adminData.FsImage{
			Image:    routes.ImageID(*i.ID),
			Name:     i.Filename + "." + i.Ext,
			URL:      template.URL(routes.CreateAdminImgPath(routes.ImageID(*i.ID))),
			LastSync: i.LastSyncDate.Format("2006.01.02 15:04:05"),
			ACL:      acl,
		}
		imageItems = append(imageItems, img)
	}
	ret := adminData.FsImages{
		Images: imageItems,
		Paging: paging,
	}
	logging.Exit(logScope, "ok", nil)
	return ret, nil

}

func createFsBreadcrumbs(root, path string, lang string, i18n *i18n.Service) tplData.Breadcrumbs {
	parts := []string{}
	if root != "" {
		parts = append(parts, root)
	}
	if path != "" {
		parts = append(parts, strings.Split(path, "/")...)
	}
	selfRoot := tplData.Breadcrumb{
		Label: i18n.T(lang, "nav.page.admin.images.short", nil),
		Type:  "page",
		Title: i18n.T(lang, "nav.page.admin.images.label", nil),
	}
	if len(parts) > 0 {
		selfRoot.Link = template.URL(routes.BuildAdminFsPath("").String())

	}
	res := tplData.Breadcrumbs{
		tpl.GetAdminMain(lang, i18n),
		selfRoot,
	}

	if len(parts) > 0 {

		for i := 0; i < len(parts)-1; i++ {
			path := strings.Join(parts[:i+1], "/")
			brc := tplData.Breadcrumb{
				Label: parts[i],
				Link:  template.URL(routes.BuildAdminFsPath(path).String()),
				Type:  "filesystem",
				Title: fmt.Sprintf("Browse: %s", parts[i]),
			}
			res = append(res, brc)
		}
		brc := tplData.Breadcrumb{
			Label: parts[len(parts)-1],
			Type:  "filesystem",
		}
		res = append(res, brc)
	}
	return res
}

func FSPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		path := c.Param("fsPath")
		path = strings.TrimPrefix(path, "/")
		dPageStr := c.DefaultQuery(routes.FolderPageParam, "1")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/fs", path, map[string]any{
			"path":           path,
			"directory_page": dPageStr,
			"image_page":     iPageStr,
		})

		dPage, err := tpl.ParsePaging(dPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid directory_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid directory page"})
			return
		}

		iPage, err := tpl.ParsePaging(iPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		url := routes.BuildAdminFsPath(path)
		url.WithFolderPaging(dPage).WithImagePaging(iPage)

		root := ""
		pathParts := strings.Split(path, "/")
		if len(pathParts) > 0 {
			root = pathParts[0]
			path = strings.Join(pathParts[1:], "/")
		}
		// todo: db path check?

		database := db.GetDatabase()
		var dirs adminData.FsDirs
		if root == "" {
			dirs, err = BuildRootDirGrid(ctx, database, dPage, *url)
		} else {
			dirs, err = BuildDirGrid(ctx, database, root, path, dPage, *url)
			if err != nil {
				logging.ExitErr(logScope, err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
		images, err := BuildImageGrid(ctx, database, root, path, iPage, *url)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		fsCtx := adminData.FsPageContext{}
		pageCtx := fsCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "fs", tplData.SurfaceAdmin)
		fsCtx.Dirs = dirs
		fsCtx.Images = images
		fsCtx.Breadcrumbs = createFsBreadcrumbs(root, path, loc, i18n)

		if err := r.RenderPage(c.Writer, "admin/fs", fsCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
