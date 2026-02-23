package data

import (
	"fmt"
	"html/template"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	grid "github.com/ignisVeneficus/lumenta/tpl/grid/data"
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

type UserContext struct {
	authData.ACLContext
	HasGuestEnabled bool
}

func (uc UserContext) EnableMenu() bool {
	return uc.HasGuestEnabled || uc.Role != authData.RoleGuest
}

type PageContext struct {
	Page PageInfo
	Site SiteInfo
	User UserContext
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
	Type  string
	Title string
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

type TagRootPageContext struct {
	PageContext
	Breadcrumbs Breadcrumbs
	TagsTree    dbo.TagsWCountTree
}

type ImageGrid struct {
	Images []*grid.GridImage
	Paging Paging
}

type ImageGridPageContext struct {
	PageContext
	Breadcrumbs Breadcrumbs
	ImageGrid   ImageGrid
	ImageTags   ImageTags
}

type ImageTags dbo.TagsTree

type TagPageContext struct {
	ImageGridPageContext
	PageTags PageTags
}

type PageTags struct {
	Tags   []PageTag
	Paging Paging
}

func (pt PageTags) Cards() []PageTag {
	return pt.Tags
}

type PageTag struct {
	Name   string
	Image  uint64
	ImgQty uint64
	URL    template.URL
}

func (pt PageTag) Description() string {
	return ""
}
func (pt PageTag) Info() string {
	return fmt.Sprintf("%d images", pt.ImgQty)
}
