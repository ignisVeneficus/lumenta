package routes

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

const (
	derivativePath = "/img/%d/%s"
	albumImagePath = "/album/%d/img/%d"
	albumPath      = "/album/%d"
	tagsRootPath   = "/tag/"
	tagPath        = "/tag/%d"
	tagImagePath   = "/tag/%d/img/%d"
)

func GetAlbumImagePath() string {
	return getPath(albumImagePath, ":aid", ":iid")
}
func CreateAlbumImagePath(aid, iid uint64) string {
	return fmt.Sprintf(albumImagePath, aid, iid)
}

func GetImageDerivativePath() string {
	return getPath(derivativePath, ":id", ":type")
}
func CreateDerivativePath(id uint64, derivative string) string {
	return fmt.Sprintf(derivativePath, id, derivative)
}

func GetAlbumPath() string {
	return getPath(albumPath, ":id")
}
func CreateAlbumPath(id int64) string {
	return fmt.Sprintf(albumPath, id)
}

func GetTagsRootPath() string {
	return tagsRootPath
}
func CreateTagsRootPath() string {
	return tagsRootPath
}

func GetTagPath() string {
	return getPath(tagPath, ":id")
}
func CreateTagPath(id uint64) string {
	return fmt.Sprintf(tagPath, id)
}
func BuildTagPath(tagId uint64) *data.URLBuilder {
	return data.NewURL(CreateTagPath(tagId))
}

func GetTagImagePath() string {
	return getPath(tagImagePath, ":tid", ":iid")
}
func CreateTagImagePath(tid, iid uint64) string {
	return fmt.Sprintf(tagImagePath, tid, iid)
}
