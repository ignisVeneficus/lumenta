package data

import (
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type Tag struct {
	ID       uint64  `json:"id"`
	Name     string  `json:"name"`
	FullName string  `json:"fullName"`
	ParentID *uint64 `json:"parent_id,omitempty"`
	Children []Tag   `json:"children,omitempty"`
}

func CreateTag(dboTag dbo.Tag, fullNameMap map[uint64]string) Tag {
	ID := uint64(0)
	if dboTag.ID != nil {
		ID = *dboTag.ID
	}
	fullName, ok := fullNameMap[ID]
	if !ok {
		fullName = ""
	}
	return Tag{
		ID:       ID,
		Name:     dboTag.Name,
		FullName: fullName,
		ParentID: dboTag.ParentID,
		Children: CreateTags(dboTag.Children, fullNameMap),
	}
}
func CreateTags(tags dbo.TagsTree, fullNameMap map[uint64]string) []Tag {
	if len(tags) == 0 {
		return nil
	}
	ret := make([]Tag, len(tags))
	for i, t := range tags {
		ret[i] = CreateTag(*t, fullNameMap)
	}
	return ret
}
