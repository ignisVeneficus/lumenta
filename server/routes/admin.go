package routes

import (
	"fmt"
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
func BuildAdminFsPath(path string) *URLBuilder {
	return NewURL(CreateAdminFsPath(path))
}

func GetAdminImgPath() string {
	return getPath(adminImgPath, ":id")
}
func CreateAdminImgPath(imageID ImageID) string {
	return AdminPrefix + fmt.Sprintf(adminImgPath, imageID)
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
func CreateAdminAlbumPath(albumID AlbumID) string {
	return AdminPrefix + fmt.Sprintf(adminAlbumIdPath, albumID)
}
func BuildAdminAlbumPath(albumID AlbumID) *URLBuilder {
	return NewURL(CreateAdminAlbumPath(albumID))
}

func GetAdminSyncRunsPath() string {
	return adminSyncRunListPath
}
func CreateAdminSyncRunListPath() string {
	return AdminPrefix + adminSyncRunListPath
}
func BuildAdminSyncRunListPath() *URLBuilder {
	return NewURL(CreateAdminSyncRunListPath())
}

func GetAdminSyncRunFilesPath() string {
	return getPath(adminSyncRunFilesPath, ":id")
}
func CreateAdminSyncRunFilesPath(syncRunID SyncRunID) string {
	return AdminPrefix + fmt.Sprintf(adminSyncRunFilesPath, syncRunID)
}
func BuildAdminSyncRunFilesPath(syncRunID SyncRunID) *URLBuilder {
	return NewURL(CreateAdminSyncRunFilesPath(syncRunID))
}

func GetAdminSyncFilesPath() string {
	return adminSyncFilesPath
}
func CreateAdminSyncFilesPath() string {
	return AdminPrefix + adminSyncFilesPath
}
func BuildAdminSyncFilesPath() *URLBuilder {
	return NewURL(CreateAdminSyncFilesPath())
}

func GetAdminSyncFilesByPathPath() string {
	return getPath(adminSyncFilesByPathPath, "*fPath")
}
func CreateAdminSyncFilesByPathPath(fullpath string) string {
	return AdminPrefix + fmt.Sprintf(adminSyncFilesByPathPath, fullpath)
}
func BuildAdminSyncFilesByPathPath(fullpath string) *URLBuilder {
	return NewURL(CreateAdminSyncFilesByPathPath(fullpath))
}

func GetAdminSyncFilePath() string {
	return getPath(adminSyncFilePath, ":id")
}
func CreateAdminSyncFilePath(syncFileID SyncFileID) string {
	return AdminPrefix + fmt.Sprintf(adminSyncFilePath, syncFileID)
}
func BuildAdminSyncFilePath(syncFileID SyncFileID) *URLBuilder {
	return NewURL(CreateAdminSyncFilePath(syncFileID))
}
