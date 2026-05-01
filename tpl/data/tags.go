package data

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
)

type ViewTreeNode struct {
	ID       uint64
	ParentID *uint64
	Children []*ViewTreeNode
	Label    string
	Notes    []string
	Path     []string
	Type     string
	URL      template.URL
}

func (tw *ViewTreeNode) GetSorting() string {
	return tw.Label
}

type FlatForrest struct {
	Nodes map[uint64]*ViewTreeNode
}
type Forrest struct {
	NodesIds map[uint64]*ViewTreeNode
	Roots    []*ViewTreeNode
}

func NewFlatForrest() *FlatForrest {
	return &FlatForrest{
		Nodes: make(map[uint64]*ViewTreeNode),
	}
}
func (f *FlatForrest) Add(nodes []ViewTreeNode) {
	for i := range nodes {
		n := nodes[i]
		copy := n
		f.Nodes[n.ID] = &copy
	}
}
func MapToViewNodes[T any](items []T, fn func(T) ViewTreeNode) []ViewTreeNode {
	result := make([]ViewTreeNode, 0, len(items))
	for _, item := range items {
		result = append(result, fn(item))
	}
	return result
}

func (f *FlatForrest) Build() *Forrest {
	// reset children
	for _, n := range f.Nodes {
		n.Children = nil
	}

	var roots []*ViewTreeNode

	nodes := make([]*ViewTreeNode, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		nodes = append(nodes, n)
	}
	utils.SortByStringKey(nodes, (*ViewTreeNode).GetSorting)

	for _, n := range nodes {
		if n.ParentID == nil {
			roots = append(roots, n)
			continue
		}

		parent, ok := f.Nodes[*n.ParentID]
		if ok {
			parent.Children = append(parent.Children, n)
		} else {
			roots = append(roots, n)
		}
	}

	return &Forrest{NodesIds: f.Nodes, Roots: roots}
}
func (f *Forrest) SetTags(types map[string][]string) {
	byLabel := make(map[string]string)
	for k, list := range types {
		for _, l := range list {
			byLabel[l] = k
		}
	}
	for _, root := range f.Roots {
		nt, ok := byLabel[root.Label]
		if ok {
			root.Type = nt
		}
	}
}
func (f *Forrest) Populate() {
	queue := make([]*ViewTreeNode, 0)

	for _, r := range f.Roots {
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.ParentID != nil {
			parent := f.NodesIds[*curr.ParentID]
			curr.Path = append([]string{}, parent.Path...)
			curr.Type = parent.Type
		}
		curr.Path = append(curr.Path, curr.Label)
		queue = append(queue, curr.Children...)
	}
}

func (f *Forrest) Set(setter func(*ViewTreeNode)) {
	for _, n := range f.NodesIds {
		setter(n)
	}
}

func ForrestFromTags(tags []*dbo.Tag, UrlBuilder func(uint64) string) Forrest {
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
	forrest := flatForrest.Build()
	forrest.Populate()
	return *forrest
}
