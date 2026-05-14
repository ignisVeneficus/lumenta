package data

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

func ForrestFromTags(tags []*dbo.Tag, UrlBuilder func(uint64) string) *data.Forest[*ViewTreeNode] {
	flatForrest := NewFlatForrest()
	mapper := func(t *dbo.Tag) ViewTreeNode {
		return ViewTreeNode{
			ID:       *t.ID,
			ParentID: t.ParentID,
			Label:    t.Name,
			URL:      template.URL(UrlBuilder(*t.ID)),
		}
	}
	tagsList := MapToViewNodes(tags, mapper)
	flatForrest.Add(tagsList)
	forest := flatForrest.Build()
	typeSetter := func(node *ViewTreeNode, data string) {
		node.Type = data
	}
	typeGetter := func(node *ViewTreeNode) string {
		return node.Type
	}
	data.Populate(forest, typeGetter, typeSetter)
	return forest
}
func SetTags(forest *data.Forest[*ViewTreeNode], types map[string][]string) {
	byLabel := make(map[string]string)
	for k, list := range types {
		for _, l := range list {
			byLabel[l] = k
		}
	}
	for _, root := range forest.Roots {
		nt, ok := byLabel[root.Label]
		if ok {
			root.Type = nt
		}
	}
}
func SetTagsMeaning(forest *data.Forest[*ViewTreeNode], tagMeaningConfig *presentation.TagMeaningConfig) {
	if tagMeaningConfig != nil {
		types := make(map[string][]string)
		for k, v := range tagMeaningConfig.MeaningMap {
			types[string(k)] = v
		}
		SetTags(forest, types)
		typeSetter := func(node *ViewTreeNode, data string) {
			if node != nil {
				node.Type = data
			}
		}
		typeGetter := func(node *ViewTreeNode) string {
			if node == nil {
				return ""
			}
			return node.Type
		}
		data.Populate(forest, typeGetter, typeSetter)
	}

}
