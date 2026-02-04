package tpl

type Option[T any] struct {
	Name  string
	Value T
}

func Get[T any](opts []Option[T], name string) (T, bool) {
	var zero T
	for _, o := range opts {
		if o.Name == name {
			return o.Value, true
		}
	}
	return zero, false
}
