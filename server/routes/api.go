package routes

import (
	"fmt"
)

const (
	ApiPrefix         = "/api"
	apiTagPath        = "/tags/%d"
	apiAlbumsRootPath = "/album"
	apiAlbumPath      = "/album/%d"
)

func GetApiTagPath() string {
	return getPath(apiTagPath, ":tid")
}
func CreateApiTagPath(tagID TagID) string {
	return fmt.Sprintf(ApiPrefix+apiTagPath, tagID)
}
func BuildApiTagPath(tagID TagID) *URLBuilder {
	return NewURL(CreateApiTagPath(tagID))
}

func GetApiAlbumsRootPath() string {
	return apiAlbumsRootPath
}
func CreateApiAlbumsRootPath() string {
	return fmt.Sprintf(ApiPrefix + apiAlbumsRootPath)
}
func BuildApiAlbumsRootPath() *URLBuilder {
	return NewURL(CreateApiAlbumsRootPath())
}

func GetApiAlbumPath() string {
	return getPath(apiAlbumPath, ":tid")
}
func CreateApiAlbumPath(albumID AlbumID) string {
	return fmt.Sprintf(ApiPrefix+apiAlbumPath, albumID)
}
func BuildApiAlbumPath(albumID AlbumID) *URLBuilder {
	return NewURL(CreateApiAlbumPath(albumID))
}
