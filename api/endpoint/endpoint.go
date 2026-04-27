package endpoint

type ListView string

const (
	ListViewFlat ListView = "flat"
	ListViewTree ListView = "tree"
)

func (v ListView) IsValid() bool {
	switch v {
	case ListViewFlat, ListViewTree:
		return true
	default:
		return false
	}
}
