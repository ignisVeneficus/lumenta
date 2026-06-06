package data

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"sort"

	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
)

type TagNode struct {
	ID       uint64
	Name     string
	ParentID *uint64
	Count    uint64
	Depth    int
	Meaning  presentation.TagMeaning
	Ordering string
	Leaf     bool
}

type TagDiscovery struct {
	IdMap      map[uint64]*TagNode
	MeaningMap map[presentation.TagMeaning][]*TagNode
}

func (td TagDiscovery) GetRandom(meaning presentation.TagMeaning, pointer string) *TagNode {
	tags, ok := td.MeaningMap[meaning]
	if !ok {
		return nil
	}
	filtered := make([]*TagNode, 0, len(tags))
	for _, t := range tags {
		if t.Leaf {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Ordering < filtered[j].Ordering
	})
	for _, t := range filtered {
		if pointer <= t.Ordering {
			return t
		}
	}
	return filtered[0]
}

func (td TagDiscovery) GetSame(tags []*dbo.Tag) map[presentation.TagMeaning][]*dbo.Tag {
	ret := make(map[presentation.TagMeaning][]*dbo.Tag)
	common := make(map[uint64]struct{})
	hasCommonDescendant := make(map[uint64]struct{})
	idmap := make(map[uint64]*dbo.Tag)

	//utils.SortByStringKey(tags, (*dbo.Tag).GetSorting)
	for _, t := range tags {
		tn, ok := td.IdMap[t.GetID()]
		if !ok {
			continue
		}
		if string(tn.Meaning) != "" {
			common[t.GetID()] = struct{}{}
			idmap[t.GetID()] = t
		}
	}
	for id := range common {
		n := td.IdMap[id]
		if n.ParentID != nil {
			hasCommonDescendant[*n.ParentID] = struct{}{}
		}
	}
	for id := range common {
		if _, ok := hasCommonDescendant[id]; !ok {
			tn := td.IdMap[id]
			t := idmap[id]
			ret[tn.Meaning] = append(ret[tn.Meaning], t)
		}
	}
	for k := range ret {
		utils.SortByStringKey(ret[k], (*dbo.Tag).GetSorting)
	}
	return ret
}

func CreateTagDiscovery(cfg *presentation.TagMeaningConfig, tags []*dbo.TagWCount) TagDiscovery {
	if cfg == nil {
		return TagDiscovery{}
	}
	stack := make([]uint64, 0, len(tags))
	tagMap := make(map[uint64]*TagNode)
	visited := make(map[uint64]struct{})
	threshold := cfg.Threshold
	for _, item := range tags {
		if item.Count >= uint64(threshold) {
			tag := TagNode{
				ID:       uint64(*item.ID),
				ParentID: (*uint64)(item.ParentID),
				Count:    item.Count,
				Leaf:     true,
				Name:     item.Name,
			}
			stack = append(stack, tag.ID)
			tagMap[tag.ID] = &tag
		}
	}
	if len(stack) == 0 {
		return TagDiscovery{}
	}
	meaningMap := make(map[presentation.TagMeaning][]*TagNode, len(stack))
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		if _, ok := visited[id]; ok {
			stack = stack[:len(stack)-1]
			continue
		}
		tag := tagMap[id]

		if tag.ParentID == nil {
			tag.Meaning = cfg.MeaningMap.GetMeaning(tag.Name)
			tag.Depth = 0
			hash := sha256.Sum256([]byte(tag.Name))
			tag.Ordering = hex.EncodeToString(hash[:])
			visited[id] = struct{}{}
			if tag.Meaning != "" {
				meaningMap[tag.Meaning] = append(meaningMap[tag.Meaning], tag)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if _, ok := visited[*tag.ParentID]; ok {
			parent := tagMap[*tag.ParentID]
			tag.Meaning = parent.Meaning
			tag.Depth = parent.Depth + 1
			hash := sha256.Sum256([]byte(tag.Name))
			tag.Ordering = hex.EncodeToString(hash[:])
			parent.Leaf = false
			visited[id] = struct{}{}
			if tag.Meaning != "" {
				meaningMap[tag.Meaning] = append(meaningMap[tag.Meaning], tag)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		stack = append(stack, *tag.ParentID)
	}
	return TagDiscovery{
		IdMap:      tagMap,
		MeaningMap: meaningMap,
	}
}

func ForestFromTags(tags []*dbo.Tag, UrlBuilder func(uint64) string) *data.Forest[*ViewTreeNode] {
	flatForest := NewFlatForest()
	mapper := func(t *dbo.Tag) ViewTreeNode {
		return ViewTreeNode{
			ID:       uint64(*t.ID),
			ParentID: (*uint64)(t.ParentID),
			Label:    t.Name,
			URL:      template.URL(UrlBuilder(uint64(*t.ID))),
		}
	}
	tagsList := MapToViewNodes(tags, mapper)
	flatForest.Add(tagsList)
	forest := flatForest.Build()
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
			types[string(k)] = v.TagRoots
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
