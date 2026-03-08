package data

import "html/template"

type Breadcrumb struct {
	Label string
	Link  template.URL
	Type  string
	Title string
}
type Breadcrumbs []Breadcrumb
