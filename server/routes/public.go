package routes

import (
	"fmt"
)

const (
	derivativePath = "/img/%d/%s"
	albumImagePath = "/album/%d/img/%d"
	albumPath      = "/album/%d"
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
