package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/rs/zerolog"
)

type AlbumID uint64

type Album struct {
	ID          AlbumID  `json:"id"`
	Name        string   `json:"name"`
	FullName    string   `json:"fullName"`
	Description string   `json:"description"`
	ParentID    *AlbumID `json:"parent_id,omitempty"`
	Children    []*Album `json:"children,omitempty"`
}

func CreateAlbum(dboAlbum dbo.Album) *Album {
	ID := AlbumID(0)
	if dboAlbum.ID != nil {
		ID = AlbumID(*dboAlbum.ID)
	}
	return &Album{
		ID:          ID,
		Name:        dboAlbum.Name,
		Description: utils.FromStringPtr(dboAlbum.Description),
		ParentID:    (*AlbumID)(dboAlbum.ParentID),
	}
}
func CreateAlbums(albums []*dbo.Album) []*Album {
	if len(albums) == 0 {
		return nil
	}
	ret := make([]*Album, len(albums))
	for i, a := range albums {
		ret[i] = CreateAlbum(*a)
	}
	return ret
}

type AlbumPatch struct {
	ParentID    Field[dbo.AlbumID] `json:"parent_id"`
	Name        Field[string]      `json:"name"`
	Description Field[string]      `json:"description"`

	Rank Field[uint64] `json:"rank"`

	CoverImageID Field[dbo.ImageID] `json:"cover_image_id"`

	ACLLevel  Field[dbo.DBACLLevel] `json:"acl_scope"`
	ACLUserID Field[dbo.UserID]     `json:"acl_user_id"`
}

func (a *AlbumPatch) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {

		FieldAlbumIDIf(e, string(definitions.AlbumFieldParentID), a.ParentID)
		FieldStringIf(e, string(definitions.AlbumFieldName), a.Name)
		FieldStringIf(e, string(definitions.AlbumFieldDescription), a.Description)
		FieldUint64If(e, string(definitions.AlbumFieldRank), a.Rank)
		FieldImageIDIf(e, string(definitions.AlbumFieldCoverImageID), a.CoverImageID)

		if a.ACLLevel.Set && a.ACLLevel.Valid {
			e.Uint64(string(definitions.AlbumFieldACLLevel), uint64(a.ACLLevel.Value))
		}

		FieldUserIDIf(e, string(definitions.AlbumFieldACLUserID), a.ACLUserID)
	}
}
func (a *Album) IsRoot() bool {
	return a.ParentID == nil
}

func (a *Album) GetID() uint64 {
	return uint64(a.ID)
}

func (a *Album) GetParentID() *uint64 {
	return (*uint64)(a.ParentID)
}

func (a *Album) ClearChildren() {
	a.Children = a.Children[:0]
}
func (a *Album) GetChildren() []*Album {
	return a.Children
}

func (a *Album) AddChild(child *Album) {
	a.Children = append(a.Children, child)
}

func (a *Album) GetSorting() string {
	return a.Name
}
func (a *Album) GetName() string {
	return a.Name
}
func (a *Album) GetPath() string {
	return a.FullName
}
func (a *Album) SetPath(path string) {
	a.FullName = path
}
