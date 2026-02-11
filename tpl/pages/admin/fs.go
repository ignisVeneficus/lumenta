package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"math"
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
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	dirPerPage    = 24
	imagePerPage  = 48
	dirPageName   = "dPage"
	imagePageName = "iPage"
)

func BuildDirGrid(ctx context.Context, database *sql.DB, pagePath string, page int, url data.URLBuilder) (data.AdminFsDirs, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildDirGrid", map[string]any{
		"path": pagePath,
		"page": page,
	})
	qty, err := dao.CountImagePathByParentPath(database, ctx, pagePath)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.AdminFsDirs{}, err
	}
	dirs, err := dao.QueryImagePathByParentPathPaged(database, ctx, pagePath, uint64((page-1)*dirPerPage), uint64(dirPerPage))
	if err != nil {
		logging.ExitErr(logg, err)
		return data.AdminFsDirs{}, err
	}
	paging := data.Paging{
		Url:     url,
		Name:    dirPageName,
		ActPage: uint64(page),
		MaxPage: uint64(math.Ceil(float64(qty) / float64(dirPerPage))),
	}
	dirItems := []data.AdminFsDir{}
	for _, d := range dirs {
		dir := data.AdminFsDir{
			Name:   path.Base(d),
			ImgQty: 0,
			URL:    template.URL(routes.CreateAdminFsPath(d).String()),
		}
		img, err := dao.GetImageByPathFirstHash(database, ctx, d)
		switch {
		case err == nil:
			dir.Image = img
		case !errors.Is(err, dao.ErrDataNotFound):
			logging.ExitErr(logg, err)
			return data.AdminFsDirs{}, err
		}
		qty, err := dao.CountImageByParentPath(database, ctx, d)
		if err != nil {
			logging.ExitErr(logg, err)
			return data.AdminFsDirs{}, err
		}
		dir.ImgQty = qty
		dirItems = append(dirItems, dir)
	}
	ret := data.AdminFsDirs{
		Directories: dirItems,
		Paging:      paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil
}
func BuildImageGrid(ctx context.Context, database *sql.DB, path string, page int, url data.URLBuilder) (data.AdminFsImages, error) {
	logg := logging.Enter(ctx, "admin.fsPage.buildImageGrid", map[string]any{
		"path": path,
		"page": page,
	})
	qty, err := dao.CountImagesByPath(database, ctx, path)
	if err != nil {
		logging.ExitErr(logg, err)
		return data.AdminFsImages{}, err
	}
	images, err := dao.QueryImageWLastSyncWUserByPathPaged(database, ctx, path, uint64((page-1)*imagePerPage), uint64(imagePerPage))
	if err != nil {
		logging.ExitErr(logg, err)
		return data.AdminFsImages{}, err
	}

	paging := data.Paging{
		Url:     url,
		Name:    imagePageName,
		ActPage: uint64(page),
		MaxPage: uint64(math.Ceil(float64(qty) / float64(imagePerPage))),
	}

	imageItems := []data.AdminFsImage{}
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
		img := data.AdminFsImage{
			Image:    *i.ID,
			Name:     i.Filename + "." + i.Ext,
			URL:      template.URL(routes.CreateAdminImgPath(*i.ID)),
			LastSync: i.LastSyncDate.Format("2006.01.02 15:04:05"),
			ACL:      acl,
		}
		imageItems = append(imageItems, img)
	}
	ret := data.AdminFsImages{
		Images: imageItems,
		Paging: paging,
	}
	logging.Exit(logg, "ok", nil)
	return ret, nil

}

func createBreadcrunch(path string) data.Breadcrumbs {
	parts := strings.Split(path, "/")
	res := data.Breadcrumbs{
		data.Breadcrumb{
			Label: "Admin",
			Link:  template.URL(routes.CreateAdminRootPath()),
		},
		data.Breadcrumb{
			Label: "Images",
			Link:  template.URL(routes.CreateAdminFsPath("").String()),
		},
	}

	for i := 0; i < len(parts); i++ {
		path := strings.Join(parts[:i+1], "/")
		brc := data.Breadcrumb{
			Label: parts[i],
			Link:  template.URL(routes.CreateAdminFsPath(path).String()),
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

		url := routes.CreateAdminFsPath(path)
		url.WithIntQuery(dirPageName, dPage).WithIntQuery(imagePageName, iPage)

		database := db.GetDatabase()
		dirs, err := BuildDirGrid(c, database, path, dPage, *url)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		images, err := BuildImageGrid(c, database, path, iPage, *url)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)

		fsCtx := data.AdminFsContext{}
		pageCtx := fsCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "fs", data.SurfaceAdmin)
		fsCtx.Dirs = dirs
		fsCtx.Images = images
		fsCtx.Breadcrumbs = createBreadcrunch(path)

		if err := r.RenderPage(c.Writer, "admin/fs", fsCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
