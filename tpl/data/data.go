package data

import (
	"html/template"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

const (
	QueryPaging = "page"
)

type Surface string

var (
	SurfacePublic Surface = "public"
	SurfaceAdmin  Surface = "admin"
)

type PageContexHolder interface {
	GetPageContext() *PageContext
}

type PageContext struct {
	Page PageInfo
	Site SiteInfo
	User authData.ACLContext
}

func (p *PageContext) GetPage() *PageContext {
	return p
}

type PageInfo struct {
	PageRole string
	Surface  Surface
}
type SiteInfo struct {
	Title       string
	Author      string
	BaseURL     string
	Description string
	Footer      FooterInfo
	Logo        string
	Headline    string
}
type FooterInfo struct {
	Note          string
	CopyrightDate string
}
type AlbumPageContex struct {
	PageContext
	Images []*gridData.GridImage
}

type Breadcrumb struct {
	Label string
	Link  template.URL
}
type Breadcrumbs []Breadcrumb

type Paging struct {
	Url     URLBuilder
	Name    string
	MaxPage uint64
	ActPage uint64
}
type CardGridData interface {
	Cards() []GridCard
	Paging() Paging
}

type GridCard interface {
	URL() template.URL
	Image() uint64
	Name() string
	Description() string
	Info() string
}
