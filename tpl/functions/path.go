package functions

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/server/routes"
)

func ImagePath(imageID routes.ImageID, derivative string) template.URL {
	return template.URL(routes.CreateDerivativePath(imageID, derivative))
}

func AlbumsRootPath() template.URL {
	return template.URL(routes.CreateAlbumsRootPath())
}

func TagPath(tagID routes.TagID) template.URL {
	return template.URL(routes.CreateTagPath(tagID))
}

func TagsRootPath() template.URL {
	return template.URL(routes.CreateTagsRootPath())
}

func AdminRootPath() template.URL {
	return template.URL(routes.CreateAdminRootPath())
}

func AdminImagePath(imageID routes.ImageID) template.URL {
	return template.URL(routes.CreateAdminImgPath(imageID))
}

func AdminAlbumsPath() template.URL {
	return template.URL(routes.CreateAdminAlbumsPath())
}
func AdminAlbumPath(albumId routes.AlbumID) template.URL {
	return template.URL(routes.CreateAdminAlbumPath(albumId))
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
func AdminSyncRunPath(syncRunID routes.SyncRunID) template.URL {
	return template.URL(routes.CreateAdminSyncRunFilesPath(syncRunID))
}
func AdminSyncFilesPathPath(path string) template.URL {
	return template.URL(routes.CreateAdminSyncFilesByPathPath(path))
}
func AdminSyncFilePath(syncFileID routes.SyncFileID) template.URL {
	return template.URL(routes.CreateAdminSyncFilePath(syncFileID))
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

func ApiAdminImagePath(imageID routes.ImageID) template.URL {
	return template.URL(routes.CreateApiAdminImagePath(imageID))
}

func ApiAdminTagsPathView(view string) template.URL {
	path := routes.BuildApiAdminTagsPath()
	if view != "" {
		path.WithParam("view", view)
	}
	return template.URL(path.String())
}
