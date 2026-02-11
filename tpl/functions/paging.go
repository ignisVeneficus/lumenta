package functions

import (
	"html/template"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func PagingFirst(paging data.Paging) template.URL {
	page := paging.ActPage
	if page > 1 {
		page = 1
	}
	url := paging.Url
	url.WithIntQuery(paging.Name, int(page))
	return template.URL(url.String())
}
func PagingPrev(paging data.Paging) template.URL {
	page := paging.ActPage
	if page > 1 {
		page--
	}
	url := paging.Url
	url.WithIntQuery(paging.Name, int(page))
	return template.URL(url.String())
}
func PagingNext(paging data.Paging) template.URL {
	page := paging.ActPage
	if page < paging.MaxPage {
		page++
	}
	url := paging.Url
	url.WithIntQuery(paging.Name, int(page))
	return template.URL(url.String())
}
func PagingLast(paging data.Paging) template.URL {
	page := paging.ActPage
	if page < paging.MaxPage {
		page = paging.MaxPage
	}
	url := paging.Url
	url.WithIntQuery(paging.Name, int(page))
	return template.URL(url.String())
}
