package routes

import (
	"fmt"
	"html/template"
)

const (
	apiAdminTagPath    = "/tags"
	apiAdminAlbumsPath = "/albums"
	apiAdminAlbumPath  = "/albums/%d"
	apiAdminImagesPath = "/images"
	apiAdminImagePath  = "/images/%d"
)

func GetApiAdminTagsPath() string {
	return apiAdminTagPath
}
func CreateApiAdminTagsPath() string {
	return ApiPrefix + AdminPrefix + apiAdminTagPath
}
func BuildApiAdminTagsPath() *URLBuilder {
	return NewURL(ApiPrefix + AdminPrefix + apiAdminTagPath)
}

func GetApiAdminAlbumsPath() string {
	return apiAdminAlbumsPath
}
func CreateApiAdminAlbumsPath() string {
	return ApiPrefix + AdminPrefix + apiAdminAlbumsPath
}
func BuildApiAdminAlbumsPath() *URLBuilder {
	return NewURL(ApiPrefix + AdminPrefix + apiAdminAlbumsPath)
}

func GetApiAdminAlbumPath() string {
	return getPath(apiAdminAlbumPath, ":id")
}
func CreateApiAdminAlbumPath(albumID AlbumID) string {
	return ApiPrefix + AdminPrefix + fmt.Sprintf(apiAdminAlbumPath, albumID)
}
func CreateApiAdminAlbumPathJS() template.JS {
	return template.JS(ApiPrefix + AdminPrefix + getPath(apiAdminAlbumPath, "{id}"))
}

func GetApiAdminImagesPath() string {
	return apiAdminImagesPath
}
func CreateApiAdminImagesPath() string {
	return ApiPrefix + AdminPrefix + apiAdminImagesPath
}
func GetApiAdminImagePath() string {
	return getPath(apiAdminImagePath, ":id")
}
func CreateApiAdminImagePath(imageID ImageID) string {
	return ApiPrefix + AdminPrefix + fmt.Sprintf(apiAdminImagePath, imageID)
}
