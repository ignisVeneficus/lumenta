package data

import (
	"fmt"
	"html/template"

	focusdata "github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	grid "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

var EMDash = "–"

type FsContext struct {
	data.PageContext
	Breadcrumbs data.Breadcrumbs
	Dirs        FsDirs
	Images      FsImages
}
type FsDirs struct {
	Directories []FsDir
	Paging      data.Paging
}

func (d FsDirs) Cards() []FsDir {
	return d.Directories
}

type FsDir struct {
	Name   string
	Image  uint64
	ImgQty uint64
	URL    template.URL
}

func (d FsDir) Description() string {
	return ""
}
func (d FsDir) Info() string {
	return fmt.Sprintf("%d images", d.ImgQty)
}

type FsImages struct {
	Images []FsImage
	Paging data.Paging
}

func (i FsImages) Cards() []FsImage {
	return i.Images
}

type FsImage struct {
	Name     string
	Image    uint64
	URL      template.URL
	ACL      string
	LastSync string
}

func (i FsImage) Description() string {
	return i.ACL
}
func (i FsImage) Info() string {
	return i.LastSync
}

type ImageContext struct {
	data.PageContext
	Breadcrumbs data.Breadcrumbs
	Image       PageImage
	Users       []dbo.User
}

type PageImage struct {
	dbo.Image
	Realpath      string
	ComputedFocus focusdata.Focus
	Aspect        grid.Aspect
	Sync          dbo.SyncRun
}

func (pi PageImage) ClampedAspect() float64 {
	ratio := pi.CalculatedAspect()
	if ratio < 1 {
		return 1
	}
	return ratio
}
func (pi PageImage) CalculatedAspect() float64 {
	return float64(pi.Width) / float64(pi.Height)
}
func (pi PageImage) CalculatedAspectString() string {
	return fmt.Sprintf("%.2f", pi.CalculatedAspect())
}
func (pi PageImage) HasTitle() bool {
	return pi.Title != nil && (*pi.Title) != ""
}
func (pi PageImage) NormalisedTitle() string {
	if pi.HasTitle() {
		return *pi.Title
	}
	return EMDash
}
func (pi PageImage) HasSubject() bool {
	return pi.Subject != nil && (*pi.Subject) != ""
}
func (pi PageImage) NormalisedSubject() string {
	if pi.HasSubject() {
		return *pi.Subject
	}
	return EMDash
}
