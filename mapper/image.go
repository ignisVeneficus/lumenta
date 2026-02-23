package mapper

import (
	"encoding/json"

	"strings"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/rs/zerolog/log"
)

func SplitTagPath(path string) []string {
	log.Logger.Debug().Str("path", path).Msg("splitPath")
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}

	return out
}
func UpdateImageMetadata(i *dbo.Image, metadata data.Metadata) error {
	i.Aperture = metadata.GetAperture()
	i.Camera = metadata.GetMakerCamera()
	i.TakenAt = metadata.GetTakenAt()
	i.FocalLength = metadata.GetFocalLength()
	i.Exposure = metadata.GetExposure()
	i.ISO = metadata.GetIso()
	i.Latitude = metadata.GetLatitude()
	i.Longitude = metadata.GetLongitude()
	i.Lens = metadata.GetLens()
	i.Rating = metadata.GetRating()
	i.Width = metadata.GetWidth()
	i.Height = metadata.GetHeight()
	i.Rotation = metadata.GetRotation()
	i.Subject = metadata.GetSubject()
	i.Title = metadata.GetTitle()
	tags := metadata.GetTags()
	JSONMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	i.ExifJSON = JSONMetadata
	i.Tags = dbo.TagsTree{}
	for _, t := range tags {
		tname := SplitTagPath(t)
		current := &dbo.Tag{Name: tname[len(tname)-1], Source: "digikam"}
		current.Children = nil
		for j := len(tname) - 2; j >= 0; j-- {
			parent := &dbo.Tag{Name: tname[j], Source: "digikam"}
			parent.Children = dbo.TagsTree{current}
			current = parent
		}
		i.Tags = append(i.Tags, current)
	}

	return nil
}

func MapACL(acl authData.ACLRole, user *uint64) *dbo.ACLScope {
	v := dbo.ACLScopeAdmin
	switch acl {
	case authData.RoleAdmin:
		v = dbo.ACLScopeAdmin
	case authData.RoleUser:
		if user != nil {
			v = dbo.ACLScopeUser
		} else {
			v = dbo.ACLScopeAnyUser
		}
	}
	return &v
}
