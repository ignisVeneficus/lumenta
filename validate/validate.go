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
		validateErrors.AddError(definitions.AlbumFieldACLLevel, "Invalid value")
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
func ValidateImage(image dbo.Image) ValidationErrors {
	validateErrors := make(ValidationErrors)
	if !dbo.ValidateACLLevel(image.ACLLevel) {
		validateErrors.AddError(definitions.ImageFieldACLLevel, "Invalid value")
	}
	if !dbo.ValidateImageFocusMode(image.FocusMode) {
		validateErrors.AddError(definitions.ImageFieldFocusMode, "Invalid value")
	}
	switch image.FocusMode {
	case dbo.ImageFocusModeManual:
		if image.FocusX == nil {
			validateErrors.AddError(definitions.ImageFieldFocusX, "Mandatory")
		} else if !dbo.ValidateImageFocus(image.FocusX) {
			validateErrors.AddError(definitions.ImageFieldFocusX, "Invalid value")
		}
		if image.FocusY == nil {
			validateErrors.AddError(definitions.ImageFieldFocusY, "Mandatory")
		} else if !dbo.ValidateImageFocus(image.FocusY) {
			validateErrors.AddError(definitions.ImageFieldFocusY, "Invalid value")
		}
	default:
		if image.FocusX != nil {
			validateErrors.AddError(definitions.ImageFieldFocusX, "Must empty")
		}
		if image.FocusY != nil {
			validateErrors.AddError(definitions.ImageFieldFocusY, "Must empty")
		}
	}
	return validateErrors

}
