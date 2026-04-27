package admin

import (
	"fmt"
	"html/template"

	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/validate"
	"github.com/rs/zerolog"
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

type AlbumForm struct {
	DBOID       *uint64 `form:"-"`
	ID          string  `form:"id"`
	Name        string  `form:"name"`
	Description string  `form:"description"`
	ParentID    string  `form:"parent_id"`
	RuleJSON    string  `form:"json_rules"`
	ACLLevel    string  `form:"acl_level"`
	ACLUserID   string  `form:"acl_user"`
	Rank        string  `form:"rank"`

	Errors validate.ValidationErrors `form:"-"`
}

func (a *AlbumForm) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str(string(definitions.AlbumFieldName), a.Name).
			Str(string(definitions.AlbumFieldID), a.ID).
			Str(string(definitions.AlbumFieldDescription), a.Description).
			Str(string(definitions.AlbumFieldParentID), a.ParentID).
			Str(string(definitions.AlbumFieldRuleJSON), a.RuleJSON).
			Str(string(definitions.AlbumFieldACLLevel), a.ACLLevel).
			Str(string(definitions.AlbumFieldACLUserID), a.ACLUserID).
			Str(string(definitions.AlbumFieldRank), a.Rank)
		errors := zerolog.Dict()
		for k, v := range a.Errors {
			errors.Strs(string(k), v)
		}
		e.Dict("Error", errors)
		logging.Uint64If(e, "DBOID", a.DBOID)
	}

}

type AlbumContext struct {
	AlbumForm
	CoverImage *uint64
	AlbumCount uint64
	ImageCount uint64
	State      FormState
	Flash      Flash
}

type AlbumPageContext struct {
	data.NavigationContext
	Album AlbumContext
}
