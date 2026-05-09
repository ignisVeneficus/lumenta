package data

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/data"
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
func (tw *ViewTreeNode) GetID() uint64 {
	return tw.ID
}
func (tw *ViewTreeNode) GetParentID() *uint64 {
	return tw.ParentID
}
func (tw *ViewTreeNode) ClearChildren() {
	tw.Children = tw.Children[0:]
}
func (tw *ViewTreeNode) AddChild(child *ViewTreeNode) {
	tw.Children = append(tw.Children, child)
}
func (tw *ViewTreeNode) GetChildren() []*ViewTreeNode {
	return tw.Children
}

type FlatForrest struct {
	Nodes map[uint64]*ViewTreeNode
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

func (f *FlatForrest) Build() *data.Forest[*ViewTreeNode] {

	nodes := make([]*ViewTreeNode, 0)
	for _, n := range f.Nodes {
		nodes = append(nodes, n)
	}
	utils.SortByStringKey(nodes, (*ViewTreeNode).GetSorting)
	return data.BuildForest(nodes)
}
