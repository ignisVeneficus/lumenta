package data

import "math"

type Paging struct {
	Url     URLBuilder
	Name    string
	MaxPage uint64
	ActPage uint64
}

func CreatePaging(url URLBuilder, name string, page int, qty uint64, perPage int) Paging {
	return Paging{
		Url:     url,
		Name:    name,
		ActPage: uint64(page),
		MaxPage: uint64(math.Ceil(float64(qty) / float64(perPage))),
	}
}
