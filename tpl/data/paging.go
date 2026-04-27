package data

import (
	"html/template"
	"math"
)

type Paging struct {
	Url     URLBuilder
	Name    string
	MaxPage uint64
	ActPage uint64
}

func (p Paging) First() template.URL {
	page := p.ActPage
	if page > 1 {
		page = 1
	}
	url := p.Url
	url.WithIntQuery(p.Name, int(page))
	return template.URL(url.String())
}
func (p Paging) Prev() template.URL {
	page := p.ActPage
	if page > 1 {
		page--
	}
	url := p.Url
	url.WithIntQuery(p.Name, int(page))
	return template.URL(url.String())
}
func (p Paging) Next() template.URL {
	page := p.ActPage
	if page < p.MaxPage {
		page++
	}
	url := p.Url
	url.WithIntQuery(p.Name, int(page))
	return template.URL(url.String())
}
func (p Paging) Last() template.URL {
	page := p.ActPage
	if page < p.MaxPage {
		page = p.MaxPage
	}
	url := p.Url
	url.WithIntQuery(p.Name, int(page))
	return template.URL(url.String())
}

func CreatePaging(url URLBuilder, name string, page, qty, perPage uint64) Paging {
	return Paging{
		Url:     url,
		Name:    name,
		ActPage: page,
		MaxPage: uint64(math.Ceil(float64(qty) / float64(perPage))),
	}
}
