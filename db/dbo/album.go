package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/rs/zerolog"
)

type Album struct {
	ID          *uint64
	ParentID    *uint64
	Name        string
	Description *string

	Rank uint64

	AncestorIDs  []uint64
	RuleJSON     json.RawMessage
	CoverImageID *uint64

	ChildAlbumCount uint32
	ImageCount      uint32

	ACLLevel  DBACLLevel
	ACLUserID uint64

	UpdatedAt time.Time

	Children []*Album

	Images []Image
}

func (a *Album) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str(string(definitions.AlbumFieldName), a.Name).
			Uint64(string(definitions.AlbumFieldACLLevel), uint64(a.ACLLevel)).
			Uint32(string(definitions.AlbumFieldChildAlbumCount), a.ChildAlbumCount).
			Uint32(string(definitions.AlbumFieldImageCount), a.ImageCount).
			Uint64(string(definitions.AlbumFieldACLUserID), a.ACLUserID)

		logging.Uint64If(e, string(definitions.AlbumFieldParentID), a.ParentID)
		logging.Uint64If(e, string(definitions.AlbumFieldCoverImageID), a.CoverImageID)
		logging.Uint64If(e, string(definitions.AlbumFieldID), a.ID)

	}
	if level == zerolog.TraceLevel {
		e.RawJSON(string(definitions.AlbumFieldRuleJSON), a.RuleJSON).
			Time(string(definitions.AlbumFieldUpdatedAt), a.UpdatedAt)
		logging.StrIf(e, string(definitions.AlbumFieldDescription), a.Description)
	}
}

func (a *Album) IsDescendantOf(parent *Album) bool {
	if len(a.AncestorIDs) < len(parent.AncestorIDs) {
		return false
	}
	for i := range parent.AncestorIDs {
		if a.AncestorIDs[i] != parent.AncestorIDs[i] {
			return false
		}
	}
	return true
}
func (a Album) GetAncestorJSON() (json.RawMessage, error) {
	return json.Marshal(a.AncestorIDs)
}

func (a *Album) AncestorPrefixJSON() string {
	if len(a.AncestorIDs) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(a.AncestorIDs)
	return string(b[:len(b)-1])
}

func (a *Album) IsRoot() bool {
	return a.ParentID == nil
}

func (a *Album) GetID() uint64 {
	return *a.ID
}

func (a *Album) GetParentID() *uint64 {
	return a.ParentID
}

func (a *Album) ClearChildren() {
	a.Children = a.Children[:0]
}

func (a *Album) AddChild(child *Album) {
	a.Children = append(a.Children, child)
}

func (a *Album) GetSorting() string {
	return a.Name
}
func (a *Album) GetPath() string {
	return a.Name
}

func ReplaceAncestorPrefix(oldAncestors, newAncestors, current []uint64) []uint64 {
	if len(current) < len(oldAncestors) {
		return current
	}
	return append(
		append([]uint64{}, newAncestors...),
		current[len(oldAncestors):]...,
	)
}
func BuildAncestorIDs(parent *Album, selfID uint64) []uint64 {
	if parent == nil {
		return []uint64{selfID}
	}

	out := make([]uint64, 0, len(parent.AncestorIDs)+1)
	out = append(out, parent.AncestorIDs...)
	out = append(out, selfID)
	return out
}

type AlbumGraph struct {
	ID       uint64
	Name     string
	ParentID *uint64
}

func (a *AlbumGraph) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name).
			Uint64("id", a.ID)
		logging.Uint64If(e, string(definitions.AlbumFieldParentID), a.ParentID)
	}
}

func (a *AlbumGraph) GetID() uint64 {
	return a.ID
}

func (a *AlbumGraph) GetParentID() *uint64 {
	return a.ParentID
}
func (a *AlbumGraph) GetName() string {
	return a.Name
}

func AlbumsToPointer(albums []Album) []*Album {
	ret := make([]*Album, len(albums))
	for i := range albums {
		ret[i] = &albums[i]
	}
	return ret
}

func AlbumGraphToPointer(albums []AlbumGraph) []*AlbumGraph {
	ret := make([]*AlbumGraph, len(albums))
	for i := range albums {
		ret[i] = &albums[i]
	}
	return ret
}
func BuildAlbumsTree(albums []*Album) []*Album {
	utils.SortByStringKey(albums, (*Album).GetSorting)
	return utils.BuildTree(albums)
}
func BuildAlbumsFlat(albums []*Album) []*Album {
	utils.SortByStringKey(albums, (*Album).GetSorting)
	for _, a := range albums {
		a.ClearChildren()
	}
	return albums
}
