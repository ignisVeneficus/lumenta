package admin

import (
	"fmt"
	"html/template"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

var EMDash = "–"

type FormState string

var (
	StateNew      FormState = "new"
	StateEdit     FormState = "edit"
	StateValidate FormState = "validate"
	StateSaved    FormState = "saved"
)

type Flash string

var (
	FlashNone    Flash = ""
	FlashCreated Flash = "created"
	FlashSaved   Flash = "saved"
	FlashDeleted Flash = "deleted"
)

type FsPageContext struct {
	data.NavigationContext
	Dirs   FsDirs
	Images FsImages
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
