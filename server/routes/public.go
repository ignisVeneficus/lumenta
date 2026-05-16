package routes

import (
	"fmt"
)

const (
	derivativePath = "/img/%d/%s"
	albumImagePath = "/album/%d/img/%d"
	albumPath      = "/album/%d"
	albumsRootPath = "/albums/"
	tagsRootPath   = "/tag/"
	tagPath        = "/tag/%d"
	tagImagePath   = "/tag/%d/img/%d"
	imagePath      = "/img/%d"
)

func GetAlbumImagePath() string {
	return getPath(albumImagePath, ":aid", ":iid")
}
func CreateAlbumImagePath(albumID AlbumID, imageID ImageID) string {
	return fmt.Sprintf(albumImagePath, albumID, imageID)
}

func GetImageDerivativePath() string {
	return getPath(derivativePath, ":id", ":type")
}
func CreateDerivativePath(id ImageID, derivative string) string {
	return fmt.Sprintf(derivativePath, id, derivative)
}

func GetAlbumsRootPath() string {
	return albumsRootPath
}
func CreateAlbumsRootPath() string {
	return albumsRootPath
}
func BuildAlbumsRootPath() *URLBuilder {
	return NewURL(albumsRootPath)
}

func GetAlbumPath() string {
	return getPath(albumPath, ":aid")
}
func CreateAlbumPath(albumID AlbumID) string {
	return fmt.Sprintf(albumPath, albumID)
}
func BuildAlbumPath(albumID AlbumID) *URLBuilder {
	return NewURL(CreateAlbumPath(albumID))
}

func GetTagsRootPath() string {
	return tagsRootPath
}
func CreateTagsRootPath() string {
	return tagsRootPath
}

func GetTagPath() string {
	return getPath(tagPath, ":tid")
}
func CreateTagPath(tagID TagID) string {
	return fmt.Sprintf(tagPath, tagID)
}
func BuildTagPath(tagID TagID) *URLBuilder {
	return NewURL(CreateTagPath(tagID))
}

func GetTagImagePath() string {
	return getPath(tagImagePath, ":tid", ":iid")
}
func CreateTagImagePath(tagID TagID, imageID ImageID) string {
	return fmt.Sprintf(tagImagePath, tagID, imageID)
}
func BuildTagImagePath(tagID TagID, imageID ImageID) *URLBuilder {
	return NewURL(CreateTagImagePath(tagID, imageID))
}
func GetImagePath() string {
	return getPath(imagePath, ":id")
}
func CreateImagePath(id ImageID) string {
	return fmt.Sprintf(imagePath, id)
}
