package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	dirPerPage    = 24
	imagePerPage  = 48
	dirPageName   = "dPage"
	imagePageName = "iPage"
)

func BuildRootDirGrid(ctx context.Context, database *sql.DB, page int, url data.URLBuilder) (adminData.FsDirs, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildRootDirGrid", map[string]any{
		"page": page,
	})
	qty, err := dao.CountImageRoots(database, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsDirs{}, err
	}
	dirs, err := dao.QueryImageRoots(database, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsDirs{}, err
	}
	paging := tpl.CreatePaging(url, dirPageName, page, qty, dirPerPage)

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
			dir.Image = img
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			return adminData.FsDirs{}, err
		}
		qty, err := dao.CountImageByRoot(database, ctx, d)
		if err != nil {
			logging.ExitErr(logg, err)
			return adminData.FsDirs{}, err
		}
		dir.ImgQty = qty
		dirItems = append(dirItems, dir)
	}
	ret := adminData.FsDirs{
		Directories: dirItems,
		Paging:      paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil
}

func BuildDirGrid(ctx context.Context, database *sql.DB, root, pagePath string, page int, url data.URLBuilder) (adminData.FsDirs, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"root": root,
		"path": pagePath,
		"page": page,
	})
	qty, err := dao.CountImagePathByParentPath(database, ctx, root, pagePath)
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsDirs{}, err
	}
	dirs, err := dao.QueryImagePathByParentPathPaged(database, ctx, root, pagePath, uint64((page-1)*dirPerPage), uint64(dirPerPage))
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsDirs{}, err
	}
	paging := tpl.CreatePaging(url, dirPageName, page, qty, dirPerPage)

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
			dir.Image = img
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			return adminData.FsDirs{}, err
		}
		qty, err := dao.CountImageByParentPath(database, ctx, root, d)
		if err != nil {
			logging.ExitErr(logg, err)
			return adminData.FsDirs{}, err
		}
		dir.ImgQty = qty
		dirItems = append(dirItems, dir)
	}
	ret := adminData.FsDirs{
		Directories: dirItems,
		Paging:      paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil
}
func BuildImageGrid(ctx context.Context, database *sql.DB, root, path string, page int, url data.URLBuilder) (adminData.FsImages, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildImageGrid", map[string]any{
		"root": root,
		"path": path,
		"page": page,
	})
	if root == "" {
		logging.Exit(logg, "no root, no image", nil)
		return adminData.FsImages{}, nil
	}
	qty, err := dao.CountImagesByPath(database, ctx, root, path)
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsImages{}, err
	}
	images, err := dao.QueryImageWLastSyncWUserByPathPaged(database, ctx, root, path, uint64((page-1)*imagePerPage), uint64(imagePerPage))
	if err != nil {
		logging.ExitErr(logg, err)
		return adminData.FsImages{}, err
	}

	paging := tpl.CreatePaging(url, imagePageName, page, qty, imagePerPage)

	imageItems := []adminData.FsImage{}
	for _, i := range images {
		acl := ""
		switch i.ACLScope {
		case dbo.ACLScopePublic:
			acl = "Public"
		case dbo.ACLScopeAnyUser:
			acl = "Users"
		case dbo.ACLScopeUser:
			acl = "User: " + utils.FromStringPtr(i.User)
		}
		img := adminData.FsImage{
			Image:    *i.ID,
			Name:     i.Filename + "." + i.Ext,
			URL:      template.URL(routes.CreateAdminImgPath(*i.ID)),
			LastSync: i.LastSyncDate.Format("2006.01.02 15:04:05"),
			ACL:      acl,
		}
		imageItems = append(imageItems, img)
	}
	ret := adminData.FsImages{
		Images: imageItems,
		Paging: paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil

}

func createFsBreadcrumbs(root, path string) data.Breadcrumbs {
	parts := []string{}
	if root != "" {
		parts = append(parts, root)
	}
	if path != "" {
		parts = append(parts, strings.Split(path, "/")...)
	}
	res := data.Breadcrumbs{
		data.Breadcrumb{
			Label: "Admin",
			Link:  template.URL(routes.CreateAdminRootPath()),
			Type:  "page",
			Title: "Open Admin Dashboard",
		},
		data.Breadcrumb{
			Label: "Images",
			Link:  template.URL(routes.BuildAdminFsPath("").String()),
			Type:  "page",
			Title: "Browse filesystem",
		},
	}
	if len(parts) > 0 {

		for i := 0; i < len(parts)-1; i++ {
			path := strings.Join(parts[:i+1], "/")
			brc := data.Breadcrumb{
				Label: parts[i],
				Link:  template.URL(routes.BuildAdminFsPath(path).String()),
				Type:  "filesystem",
				Title: fmt.Sprintf("Browse: %s", parts[i]),
			}
			res = append(res, brc)
		}
		brc := data.Breadcrumb{
			Label: parts[len(parts)-1],
			Type:  "filesystem",
		}
		res = append(res, brc)
	}
	return res
}

func FSPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("fsPath")
		path = strings.TrimPrefix(path, "/")
		dPageStr := c.DefaultQuery(dirPageName, "1")
		iPageStr := c.DefaultQuery(imagePageName, "1")
		logg := logging.Enter(c, "admin.fsPage", map[string]any{
			"path":           path,
			"directory_page": dPageStr,
			"image_page":     iPageStr,
		})

		dPage, err := strconv.Atoi(dPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid directory_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid directory page"})
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image page"})
			return
		}

		url := routes.BuildAdminFsPath(path)
		url.WithIntQuery(dirPageName, dPage).WithIntQuery(imagePageName, iPage)

		root := ""
		pathParts := strings.Split(path, "/")
		if len(pathParts) > 0 {
			root = pathParts[0]
			path = strings.Join(pathParts[1:], "/")
		}

		database := db.GetDatabase()
		var dirs adminData.FsDirs
		if root == "" {
			dirs, err = BuildRootDirGrid(c, database, dPage, *url)
		} else {
			dirs, err = BuildDirGrid(c, database, root, path, dPage, *url)
			if err != nil {
				logging.ExitErr(logg, err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}
		images, err := BuildImageGrid(c, database, root, path, iPage, *url)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)

		fsCtx := adminData.FsContext{}
		pageCtx := fsCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "fs", data.SurfaceAdmin)
		fsCtx.Dirs = dirs
		fsCtx.Images = images
		fsCtx.Breadcrumbs = createFsBreadcrumbs(root, path)

		if err := r.RenderPage(c.Writer, "admin/fs", fsCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
