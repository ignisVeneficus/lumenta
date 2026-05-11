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

type Tag struct {
	ID       *uint64
	Name     string
	ParentID *uint64
	Source   TagSource
	Children TagsTree
}

func (t *Tag) GetID() uint64 {
	return *t.ID
}

func (a *Tag) GetParentID() *uint64 {
	return a.ParentID
}

func (t *Tag) GetSorting() string {
	return t.Name
}

func (a *Tag) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name)
		logging.Uint64If(e, "id", a.ID)
		logging.Uint64If(e, "parent_id", a.ParentID)
	}
	if level <= zerolog.TraceLevel {
		e.Str("source", string(a.Source))
	}
}

type TagWCount struct {
	ID       *uint64
	Name     string
	ParentID *uint64
	Source   TagSource
	Count    uint64
}

func (t *TagWCount) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", t.Name).
			Uint64("count", t.Count)
		logging.Uint64If(e, "id", t.ID)
		logging.Uint64If(e, "parent_id", t.ParentID)
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
