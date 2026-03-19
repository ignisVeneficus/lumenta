package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
)

type ImageCoord struct {
	Image     uint64  `json:"img"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Color     string  `json:"color"`
	URL       string  `json:"url"`
	ImageURL  string  `json:"imgUrl"`
	Label     string  `json:"label"`
}

func CreateImageCoord(img dbo.ImageCoord, creator func(uint64) string) ImageCoord {
	return ImageCoord{
		Image:     img.ID,
		Latitude:  *img.Latitude,
		Longitude: *img.Longitude,
		Color:     "primary",
		URL:       creator(img.ID),
		ImageURL:  routes.CreateDerivativePath(img.ID, "s400"),
		Label:     img.Title,
	}
}
