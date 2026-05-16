package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/rs/zerolog"
)

type AlbumID uint64

type AncestorIDs []AlbumID

type Album struct {
	ID          *AlbumID
	ParentID    *AlbumID
	Name        string
	Description *string

	Rank uint64

	AncestorIDs  AncestorIDs
	RuleJSON     json.RawMessage
	CoverImageID *ImageID

	ACLLevel  DBACLLevel
	ACLUserID UserID

	UpdatedAt time.Time

	Images []Image
}

func (a *Album) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str(string(definitions.AlbumFieldName), a.Name).
			Uint64(string(definitions.AlbumFieldACLLevel), uint64(a.ACLLevel)).
			Uint64(string(definitions.AlbumFieldACLUserID), uint64(a.ACLUserID))
		logging.Uint64If(e, string(definitions.AlbumFieldParentID), (*uint64)(a.ParentID))
		logging.Uint64If(e, string(definitions.AlbumFieldID), (*uint64)(a.ID))
	}
	if level == zerolog.TraceLevel {
		e.RawJSON(string(definitions.AlbumFieldRuleJSON), a.RuleJSON).
			Time(string(definitions.AlbumFieldUpdatedAt), a.UpdatedAt)
		logging.StrIf(e, string(definitions.AlbumFieldDescription), a.Description)
		logging.Uint64If(e, string(definitions.AlbumFieldCoverImageID), (*uint64)(a.CoverImageID))

	}
}

func (a AncestorIDs) IsDescendantOf(parent AncestorIDs) bool {
	if len(a) < len(parent) {
		return false
	}
	for i := range parent {
		if a[i] != parent[i] {
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

func (a *Album) GetSorting() string {
	return a.Name
}

func ReplaceAncestorPrefix(oldAncestors, newAncestors, current AncestorIDs) AncestorIDs {
	if len(current) < len(oldAncestors) {
		return current
	}
	return append(
		append([]AlbumID{}, newAncestors...),
		current[len(oldAncestors):]...,
	)
}
func BuildAncestorIDs(parent *Album, selfID AlbumID) AncestorIDs {
	if parent == nil {
		return []AlbumID{selfID}
	}

	out := make([]AlbumID, 0, len(parent.AncestorIDs)+1)
	out = append(out, parent.AncestorIDs...)
	out = append(out, selfID)
	return out
}

type AlbumGraph struct {
	ID       AlbumID
	Name     string
	ParentID *AlbumID
}

func (a *AlbumGraph) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name).
			Uint64("id", uint64(a.ID))
		logging.Uint64If(e, string(definitions.AlbumFieldParentID), (*uint64)(a.ParentID))
	}
}

func (a *AlbumGraph) GetID() uint64 {
	return uint64(a.ID)
}

func (a *AlbumGraph) GetParentID() *uint64 {
	return (*uint64)(a.ParentID)
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
