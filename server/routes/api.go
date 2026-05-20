package routes

import (
	"fmt"
)

const (
	ApiPrefix              = "/api"
	apiTagCoordPath        = "/tags/%d/coord"
	apiAlbumsRootCoordPath = "/albums/coord"
	apiAlbumCoordPath      = "/album/%d/coord"
)

func GetApiTagCoordPath() string {
	return getPath(apiTagCoordPath, ":tid")
}
func CreateApiTagCoordPath(tagID TagID) string {
	return fmt.Sprintf(ApiPrefix+apiTagCoordPath, tagID)
}
func BuildApiTagCoordPath(tagID TagID) *URLBuilder {
	return NewURL(CreateApiTagCoordPath(tagID))
}

func GetApiAlbumsRootCoordPath() string {
	return apiAlbumsRootCoordPath
}
func CreateApiAlbumsRootCoordPath() string {
	return fmt.Sprintf(ApiPrefix + apiAlbumsRootCoordPath)
}
func BuildApiAlbumsRootCoordPath() *URLBuilder {
	return NewURL(CreateApiAlbumsRootCoordPath())
}

func GetApiAlbumCoordPath() string {
	return getPath(apiAlbumCoordPath, ":tid")
}
func CreateApiAlbumCoordPath(albumID AlbumID) string {
	return fmt.Sprintf(ApiPrefix+apiAlbumCoordPath, albumID)
}
func BuildApiAlbumCoordPath(albumID AlbumID) *URLBuilder {
	return NewURL(CreateApiAlbumCoordPath(albumID))
}
