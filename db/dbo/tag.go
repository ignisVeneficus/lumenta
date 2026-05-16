package dbo

import (
	"github.com/ignisVeneficus/logging"
	"github.com/rs/zerolog"
)

//
// =========================================================
// TAGS
// =========================================================
//

type TagsTree []*Tag

func (tt TagsTree) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	array := zerolog.Arr()
	for _, t := range tt {
		array.Object(logging.WithLevel(level, t))
	}
	e.Array("tags", array)
}

type TagID uint64

type Tag struct {
	ID       *TagID
	Name     string
	ParentID *TagID
	Source   TagSource
	Children TagsTree
}

func (t *Tag) GetID() uint64 {
	return uint64(*t.ID)
}

func (a *Tag) GetParentID() *uint64 {
	return (*uint64)(a.ParentID)
}

func (t *Tag) GetSorting() string {
	return t.Name
}

func (a *Tag) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name)
		logging.Uint64If(e, "id", (*uint64)(a.ID))
		logging.Uint64If(e, "parent_id", (*uint64)(a.ParentID))
	}
	if level <= zerolog.TraceLevel {
		e.Str("source", string(a.Source))
	}
}

type TagWCount struct {
	ID       *TagID
	Name     string
	ParentID *TagID
	Source   TagSource
	Count    uint64
}

func (t *TagWCount) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", t.Name).
			Uint64("count", t.Count)
		logging.Uint64If(e, "id", (*uint64)(t.ID))
		logging.Uint64If(e, "parent_id", (*uint64)(t.ParentID))
	}
	if level <= zerolog.TraceLevel {
		e.Str("source", string(t.Source))
	}
}

func TagsToPointer(tags []Tag) []*Tag {
	ret := make([]*Tag, len(tags))
	for i := range tags {
		ret[i] = &tags[i]
	}
	return ret
}
