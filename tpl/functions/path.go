package functions

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/server/routes"
)

func ImagePath(imageId uint64, derivative string) template.URL {
	return template.URL(routes.CreateDerivativePath(imageId, derivative))
}

func AlbumsRootPath() template.URL {
	return template.URL(routes.CreateAlbumsRootPath())
}

func TagPath(tagId uint64) template.URL {
	return template.URL(routes.CreateTagPath(uint64(tagId)))
}

func TagsRootPath() template.URL {
	return template.URL(routes.CreateTagsRootPath())
}

func AdminRootPath() template.URL {
	return template.URL(routes.CreateAdminRootPath())
}

func AdminImagePath(img uint64) template.URL {
	return template.URL(routes.CreateAdminImgPath(img))
}

func AdminAlbumsPath() template.URL {
	return template.URL(routes.CreateAdminAlbumsPath())
}

func AdminAlbumNewPath() template.URL {
	return template.URL(routes.CreateAdminAlbumNewPath())
}

func AdminFsPath() template.URL {
	return template.URL(routes.BuildAdminFsPath("").String())
}

func AdminSyncRunsPath() template.URL {
	return template.URL(routes.CreateAdminSyncRunListPath())
}
func AdminSyncRunPath(id uint64) template.URL {
	return template.URL(routes.CreateAdminSyncRunFilesPath(id))
}
func AdminSyncFilesPathPath(path string) template.URL {
	return template.URL(routes.CreateAdminSyncFilesByPathPath(path))
}
func AdminSyncFilePath(id uint64) template.URL {
	return template.URL(routes.CreateAdminSyncFilePath(id))
}
func AdminSyncFilesPath() template.URL {
	return template.URL(routes.CreateAdminSyncFilesPath())
}
func ApiAdminAlbumPathJS() template.JS {
	return routes.CreateApiAdminAlbumPathJS()
}
func ApiAdminAlbumsPathView(view string) template.URL {
	path := routes.BuildApiAdminAlbumsPath()
	if view != "" {
		path.WithParam("view", view)
	}
	return template.URL(path.String())
}

func ApiAdminImagePath(img uint64) template.URL {
	return template.URL(routes.CreateApiAdminImagePath(img))
}

func ApiAdminTagsPathView(view string) template.URL {
	path := routes.BuildApiAdminTagsPath()
	if view != "" {
		path.WithParam("view", view)
	}
	return template.URL(path.String())
}
