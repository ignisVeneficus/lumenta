package data

import (
	"errors"
	"fmt"
	"html/template"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	grid "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

var ThumbnailPerPage = 5

var ErrMissingMandatoryValue = errors.New("missing value")
var ErrInvalidValue = errors.New("invalid value")

const (
	QueryPaging = "page"
)

type Surface string

var (
	SurfacePublic Surface = "public"
	SurfaceAdmin  Surface = "admin"
)

type PageContexHolder interface {
	GetPage() *PageContext
}

type UserContext struct {
	authData.ACLContext
	HasGuestEnabled bool
}

func (uc UserContext) EnableMenu() bool {
	return uc.HasGuestEnabled || uc.Role != dbo.RoleGuest
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

type MainPageContext struct {
	PageContext
	ImageGrid ImageGrid
}

type NavigationContext struct {
	PageContext
	Breadcrumbs Breadcrumbs
}

type TagRootPageContext struct {
	NavigationContext
	TagsTree dbo.TagsWCountTree
}

type ImageGrid struct {
	Images []*grid.GridImage
	Paging *Paging
}

type ImageGridPageContext struct {
	NavigationContext
	ImageGrid ImageGrid
	ImageTags ImageTags
}

type ImageTags dbo.TagsTree

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

type ErrorPageContext struct {
	PageContext
	Resource   string
	ResourceID uint64
	RootURL    string
	Title      string
	Message    string
	BackLabel  string
}

type SingleMap struct {
	Lat   float64
	Long  float64
	Color *string
}

type ImagePageContext struct {
	NavigationContext
	Image      PageImage
	Thumbnails *Thumbnails
	Next       *Thumbnail
	Prev       *Thumbnail
	Up         string
}
