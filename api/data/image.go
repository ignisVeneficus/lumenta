package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/rs/zerolog"
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

type ImagePatch struct {
	ACLLevel  Field[dbo.DBACLLevel]     `json:"acl_scope"`
	ACLUserID Field[dbo.UserID]         `json:"acl_user_id"`
	FocusMode Field[dbo.ImageFocusMode] `json:"focus"`
	FocusX    Field[float32]            `json:"focus_x"`
	FocusY    Field[float32]            `json:"focus_y"`
}

func (i *ImagePatch) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {

		if i.ACLLevel.Set && i.ACLLevel.Valid {
			e.Uint64(string(definitions.ImageFieldACLLevel), uint64(i.ACLLevel.Value))
		}

		FieldUserIDIf(e, string(definitions.ImageFieldACLUserID), i.ACLUserID)
	}
}
