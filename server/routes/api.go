package routes

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/tpl/data"
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
func CreateApiTagPath(id uint64) string {
	return fmt.Sprintf(ApiPrefix+apiTagPath, id)
}
func BuildApiTagPath(tagId uint64) *data.URLBuilder {
	return data.NewURL(CreateApiTagPath(tagId))
}

func GetApiAlbumsRootPath() string {
	return apiAlbumsRootPath
}
func CreateApiAlbumsRootPath() string {
	return fmt.Sprintf(ApiPrefix + apiAlbumsRootPath)
}
func BuildApiAlbumsRootPath() *data.URLBuilder {
	return data.NewURL(CreateApiAlbumsRootPath())
}

func GetApiAlbumPath() string {
	return getPath(apiAlbumPath, ":tid")
}
func CreateApiAlbumPath(id uint64) string {
	return fmt.Sprintf(ApiPrefix+apiAlbumPath, id)
}
func BuildApiAlbumPath(tagId uint64) *data.URLBuilder {
	return data.NewURL(CreateApiAlbumPath(tagId))
}
