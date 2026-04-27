package utils

type TreeNode[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	ClearChildren()
	AddChild(child T)
	GetPath() string
}
type Hierarchical[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	GetName() string
}

func BuildTree[T TreeNode[T]](items []T) []T {
	tm := make(map[uint64]T, len(items))
	var roots []T

	for i := range items {
		item := items[i]
		tm[item.GetID()] = item
		item.ClearChildren()
	}

	for _, item := range items {
		if item.GetParentID() == nil {
			roots = append(roots, item)
			continue
		}
		if p, ok := tm[*item.GetParentID()]; ok {
			p.AddChild(item)
		}
	}

	return roots
}

func BuildPath[T TreeNode[T]](items []T) map[uint64]string {
	byID := make(map[uint64]T, len(items))
	for _, t := range items {
		byID[t.GetID()] = t
	}

	paths := make(map[uint64]string, len(items))

	var build func(uint64) string

	build = func(id uint64) string {
		if p, ok := paths[id]; ok {
			return p
		}
		t := byID[id]
		if t.GetParentID() == nil {
			paths[id] = t.GetPath()
			return t.GetPath()
		}
		parentPath := build(*t.GetParentID())
		p := parentPath + "/" + t.GetPath()

		paths[id] = p
		return p
	}

	for _, t := range items {
		build(t.GetID())
	}
	return paths
}

func CircleCheck[T Hierarchical[T]](items []T, start T) (string, bool) {
	byID := make(map[uint64]T, len(items))
	for _, t := range items {
		byID[t.GetID()] = t
	}
	byID[start.GetID()] = start
	current := start

	for {
		pid := current.GetParentID()
		if pid == nil {
			return "", false
		}

		if *pid == start.GetID() {
			return byID[current.GetID()].GetName(), true
		}

		next, ok := byID[*pid]
		if !ok {
			return "", false
		}
		current = next
	}
}
