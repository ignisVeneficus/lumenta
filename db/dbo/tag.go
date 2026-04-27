package dbo

import (
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"
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

func MergeTagsTrees(trees ...TagsTree) TagsTree {
	idMap := map[uint64]*Tag{}
	flat := []*Tag{}
	queu := []*Tag{}
	for _, tree := range trees {
		queu = append(queu, tree...)
	}
	for len(queu) > 0 {
		tag := queu[len(queu)-1]
		queu = queu[:len(queu)-1]
		id := tag.ID
		_, found := idMap[*id]
		if !found {
			cloned := &Tag{
				ID:       tag.ID,
				ParentID: tag.ParentID,
				Name:     tag.Name,
				Children: TagsTree{},
			}
			idMap[*id] = cloned
			queu = append(queu, tag.Children...)
			flat = append(flat, tag)
		} else {
			queu = append(queu, tag.Children...)
		}
	}
	utils.SortByStringKey(flat, (*Tag).GetSorting)
	ret := TagsTree{}
	for _, tag := range flat {
		t := idMap[*tag.ID]
		if t.ParentID == nil {
			ret = append(ret, t)
		} else {
			f, ok := idMap[*t.ParentID]
			if !ok {
				ret = append(ret, t)
			} else {
				f.Children = append(f.Children, t)
			}
		}
	}
	return ret
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

func (t *Tag) ClearChildren() {
	t.Children = t.Children[:0]
}

func (t *Tag) AddChild(child *Tag) {
	t.Children = append(t.Children, child)
}

func (t *Tag) GetSorting() string {
	return t.Name
}
func (t *Tag) GetPath() string {
	return t.Name
}

func (a *Tag) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name).
			Str("source", string(a.Source))
		logging.Uint64If(e, "id", a.ID)
		logging.Uint64If(e, "parent_id", a.ParentID)
	}
}

func BuildTagsTree(tags []*Tag) TagsTree {
	utils.SortByStringKey(tags, (*Tag).GetSorting)
	ret := utils.BuildTree(tags)
	return ret
}
func BuildTagsFlat(tags []*Tag) TagsTree {
	utils.SortByStringKey(tags, (*Tag).GetSorting)
	for _, t := range tags {
		t.ClearChildren()
	}
	return tags
}

type TagsWCountTree []*TagWCount

func (tt TagsWCountTree) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	array := zerolog.Arr()
	for _, t := range tt {
		array.Object(logging.WithLevel(level, t))
	}
	e.Array("tags", array)
}

type TagWCount struct {
	ID       *uint64
	Name     string
	ParentID *uint64
	Source   TagSource
	Count    uint64
	Children TagsWCountTree
}

func (t *TagWCount) GetID() uint64 {
	return *t.ID
}

func (t *TagWCount) GetParentID() *uint64 {
	return t.ParentID
}

func (t *TagWCount) ClearChildren() {
	t.Children = t.Children[:0]
}

func (t *TagWCount) AddChild(child *TagWCount) {
	t.Children = append(t.Children, child)
}

func (t *TagWCount) GetPath() string {
	return t.Name
}

func (t *TagWCount) GetSorting() string {
	return t.Name
}

func (t *TagWCount) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", t.Name).
			Str("source", string(t.Source)).
			Uint64("count", t.Count)
		logging.Uint64If(e, "id", t.ID)
		logging.Uint64If(e, "parent_id", t.ParentID)
	}
}

func BuildTagsWCountTree(tags []*TagWCount) TagsWCountTree {
	utils.SortByStringKey(tags, (*TagWCount).GetSorting)
	ret := utils.BuildTree(tags)
	return ret
}

func TagsToPointer(tags []Tag) []*Tag {
	ret := make([]*Tag, len(tags))
	for i := range tags {
		ret[i] = &tags[i]
	}
	return ret
}
func TagsWCountToPointer(tags []TagWCount) []*TagWCount {
	ret := make([]*TagWCount, len(tags))
	for i := range tags {
		ret[i] = &tags[i]
	}
	return ret
}
