package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type TagID uint64
type Tag struct {
	ID       TagID  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"fullName"`
	ParentID *TagID `json:"parent_id,omitempty"`
	Children []*Tag `json:"children,omitempty"`
}

func CreateTag(dboTag dbo.Tag) *Tag {
	ID := uint64(0)
	if dboTag.ID != nil {
		ID = uint64(*dboTag.ID)
	}
	return &Tag{
		ID:       TagID(ID),
		Name:     dboTag.Name,
		ParentID: (*TagID)(dboTag.ParentID),
	}
}
func CreateTags(tags dbo.TagsTree) []*Tag {
	if len(tags) == 0 {
		return nil
	}
	ret := make([]*Tag, len(tags))
	for i, t := range tags {
		ret[i] = CreateTag(*t)
	}
	return ret
}
func (t *Tag) IsRoot() bool {
	return t.ParentID == nil
}

func (t *Tag) GetID() uint64 {
	return uint64(t.ID)
}

func (t *Tag) GetParentID() *uint64 {
	return (*uint64)(t.ParentID)
}

func (t *Tag) ClearChildren() {
	t.Children = t.Children[:0]
}
func (t *Tag) GetChildren() []*Tag {
	return t.Children
}

func (t *Tag) AddChild(child *Tag) {
	t.Children = append(t.Children, child)
}

func (t *Tag) GetSorting() string {
	return t.Name
}
func (t *Tag) GetName() string {
	return t.Name
}
func (t *Tag) GetPath() string {
	return t.FullName
}
func (t *Tag) SetPath(path string) {
	t.FullName = path
}
