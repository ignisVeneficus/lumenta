package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
)

type ImageID uint64
type ImageCoord struct {
	Image     ImageID `json:"img"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Color     string  `json:"color"`
	URL       string  `json:"url"`
	ImageURL  string  `json:"imgUrl"`
	Label     string  `json:"label"`
}

func CreateImageCoord(img dbo.ImageCoord, creator func(uint64) string) ImageCoord {
	return ImageCoord{
		Image:     ImageID(img.ID),
		Latitude:  *img.Latitude,
		Longitude: *img.Longitude,
		Color:     "primary",
		URL:       creator(uint64(img.ID)),
		ImageURL:  routes.CreateDerivativePath(routes.ImageID(img.ID), "s400"),
		Label:     img.Title,
	}
}
