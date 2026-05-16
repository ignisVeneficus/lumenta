package validate

import (
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/definitions"
)

type ValidationErrors map[definitions.FieldName][]string

func (ve ValidationErrors) AddError(field definitions.FieldName, msg string) {
	ve[field] = append(ve[field], msg)
}
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}
func (ve ValidationErrors) Has(field string) bool {
	return len(ve[definitions.FieldName(field)]) > 0
}

func ValidateAlbum(album dbo.Album, albums []*dbo.AlbumGraph) ValidationErrors {
	validateErrors := make(ValidationErrors)
	if album.Name == "" {
		validateErrors.AddError(definitions.AlbumFieldName, "The field is mandatory")
	}
	if album.ID != nil && album.ID == album.ParentID {
		validateErrors.AddError(definitions.AlbumFieldParentID, "Parent cannot be the album itself.")
	}
	if !dbo.ValidateACLLevel(album.ACLLevel) {
		validateErrors.AddError(definitions.AlbumFieldACLLevel, "Invalud value")
	}
	if album.ParentID != nil && album.ParentID != album.ID {
		var albumId dbo.AlbumID = 0
		if album.ID != nil {
			albumId = *album.ID
		}
		checkAlbum := dbo.AlbumGraph{
			ID:       albumId,
			ParentID: album.ParentID,
		}
		_, circle := data.CircleCheck(albums, &checkAlbum)
		if circle {
			validateErrors.AddError(definitions.AlbumFieldParentID, "Invalid parent: circular reference detected.")

		}
	}
	return validateErrors
}
