package data

type TreeNode[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	ClearChildren()
	AddChild(child T)
	GetChildren() []T
}
type Forest[T any] struct {
	Index map[uint64]T
	Roots []T
}

func BuildForest[T TreeNode[T]](items []T) *Forest[T] {
	index := make(map[uint64]T, len(items))
	var roots []T

	for _, item := range items {
		index[item.GetID()] = item
		item.ClearChildren()
	}

	for _, item := range items {
		pid := item.GetParentID()

		if pid == nil {
			roots = append(roots, item)
			continue
		}

		if parent, ok := index[*pid]; ok {
			parent.AddChild(item)
		} else {
			roots = append(roots, item)
		}
	}

	return &Forest[T]{
		Index: index,
		Roots: roots,
	}
}

type PathTreeNode[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	GetChildren() []T
	GetName() string
	GetPath() string
	SetPath(string)
}
type FlatPathNode[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	GetName() string
	GetPath() string
	SetPath(string)
}

func PopulatePath[T PathTreeNode[T]](forest *Forest[T]) {
	queue := make([]T, 0)

	for _, r := range forest.Roots {
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.GetParentID() != nil {
			parent := forest.Index[*curr.GetParentID()]
			curr.SetPath(parent.GetPath() + "/" + curr.GetName())
		} else {
			curr.SetPath(curr.GetName())
		}
		queue = append(queue, curr.GetChildren()...)
	}
}

func Populate[T TreeNode[T], U any](forest *Forest[T], getter func(T) U, setter func(T, U)) {
	queue := make([]T, 0)

	for _, r := range forest.Roots {
		queue = append(queue, r)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.GetParentID() != nil {
			parent := forest.Index[*curr.GetParentID()]
			setter(curr, getter(parent))
		}
		queue = append(queue, curr.GetChildren()...)
	}

}

type Hierarchical[T any] interface {
	GetID() uint64
	GetParentID() *uint64
	GetName() string
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
func BuildFlatPath[T FlatPathNode[T]](nodes []T) {
	index := make(map[uint64]T)
	for i := range nodes {
		index[nodes[i].GetID()] = nodes[i]
	}

	cache := make(map[uint64]struct{})

	stack := make([]T, 0, len(nodes))
	for i := range nodes {
		stack = append(stack, nodes[i])
	}

	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		id := n.GetID()
		if _, ok := cache[id]; ok {
			continue
		}
		parentId := n.GetParentID()
		if parentId == nil {
			n.SetPath(n.GetName())
			continue
		}

		parent, ok := index[*parentId]
		if !ok {
			continue
		}
		if _, ok := cache[*parentId]; ok {
			n.SetPath(parent.GetPath() + "/" + n.GetName())
			cache[id] = struct{}{}
		} else {
			stack = append(stack, n)
			stack = append(stack, parent)
		}
	}

}
