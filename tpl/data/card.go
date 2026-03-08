package data

import "html/template"

type GridCard interface {
	URL() template.URL
	Image() uint64
	Name() string
	Description() string
	Info() string
}

type CardGridData interface {
	Cards() []GridCard
	Paging() Paging
}
