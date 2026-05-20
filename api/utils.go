package api

import (
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/utils"
)

func ParseAlbumID(s string) (data.AlbumID, error) {
	id, err := utils.ParseUint(s)
	return data.AlbumID(id), err
}
func ParseImageID(s string) (data.ImageID, error) {
	id, err := utils.ParseUint(s)
	return data.ImageID(id), err
}
func ParseTagID(s string) (data.TagID, error) {
	id, err := utils.ParseUint(s)
	return data.TagID(id), err
}
