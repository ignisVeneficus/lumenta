package routes

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

const (
	AdminPrefix  = "/admin"
	adminRoot    = "/"
	adminFsPath  = "/fs/%s"
	adminImgPath = "/img/%d"

	adminAlbumListPath = "/albums"
	adminAlbumNewPath  = "/albums/new"
	adminAlbumIdPath   = "/albums/%d"

	adminSyncRunListPath  = "/sync-runs"
	adminSyncRunFilesPath = "/sync-runs/%d"

	adminSyncFilesPath       = "/sync-files"
	adminSyncFilesByPathPath = "/sync-files/path/%s"
	adminSyncFilePath        = "/sync-file/%d"

	QueryFlash = "flash"
)

func GetAdminRootPath() string {
	return adminRoot
}
func CreateAdminRootPath() string {
	return AdminPrefix + adminRoot
}

func GetAdminFsPath() string {
	return getPath(adminFsPath, "*fsPath")
}
func CreateAdminFsPath(path string) string {
	return AdminPrefix + fmt.Sprintf(adminFsPath, path)
}
func BuildAdminFsPath(path string) *data.URLBuilder {
	return data.NewURL(CreateAdminFsPath(path))
}

func GetAdminImgPath() string {
	return getPath(adminImgPath, ":id")
}
func CreateAdminImgPath(imgID uint64) string {
	return AdminPrefix + fmt.Sprintf(adminImgPath, imgID)
}

func GetAdminAlbumsPath() string {
	return adminAlbumListPath
}
func CreateAdminAlbumsPath() string {
	return AdminPrefix + adminAlbumListPath
}

func GetAdminAlbumNewPath() string {
	return adminAlbumNewPath
}
func CreateAdminAlbumNewPath() string {
	return AdminPrefix + adminAlbumNewPath
}

func GetAdminAlbumPath() string {
	return getPath(adminAlbumIdPath, ":id")
}
func CreateAdminAlbumPath(albumID uint64) string {
	return AdminPrefix + fmt.Sprintf(adminAlbumIdPath, albumID)
}
func BuildAdminAlbumPath(albumID uint64) *data.URLBuilder {
	return data.NewURL(CreateAdminAlbumPath(albumID))
}

func GetAdminSyncRunsPath() string {
	return adminSyncRunListPath
}
func CreateAdminSyncRunListPath() string {
	return AdminPrefix + adminSyncRunListPath
}
func BuildAdminSyncRunListPath() *data.URLBuilder {
	return data.NewURL(CreateAdminSyncRunListPath())
}

func GetAdminSyncRunFilesPath() string {
	return getPath(adminSyncRunFilesPath, ":id")
}
func CreateAdminSyncRunFilesPath(id uint64) string {
	return AdminPrefix + fmt.Sprintf(adminSyncRunFilesPath, id)
}
func BuildAdminSyncRunFilesPath(id uint64) *data.URLBuilder {
	return data.NewURL(CreateAdminSyncRunFilesPath(id))
}

func GetAdminSyncFilesPath() string {
	return adminSyncFilesPath
}
func CreateAdminSyncFilesPath() string {
	return AdminPrefix + adminSyncFilesPath
}
func BuildAdminSyncFilesPath() *data.URLBuilder {
	return data.NewURL(CreateAdminSyncFilesPath())
}

func GetAdminSyncFilesByPathPath() string {
	return getPath(adminSyncFilesByPathPath, "*fPath")
}
func CreateAdminSyncFilesByPathPath(fullpath string) string {
	return AdminPrefix + fmt.Sprintf(adminSyncFilesByPathPath, fullpath)
}
func BuildAdminSyncFilesByPathPath(fullpath string) *data.URLBuilder {
	return data.NewURL(CreateAdminSyncFilesByPathPath(fullpath))
}

func GetAdminSyncFilePath() string {
	return getPath(adminSyncFilePath, ":id")
}
func CreateAdminSyncFilePath(id uint64) string {
	return AdminPrefix + fmt.Sprintf(adminSyncFilePath, id)
}
func BuildAdminSyncFilePath(id uint64) *data.URLBuilder {
	return data.NewURL(CreateAdminSyncFilePath(id))
}
