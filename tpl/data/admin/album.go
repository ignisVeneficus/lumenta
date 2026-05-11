package admin

import (
	"html/template"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/server/routes"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/validate"
	"github.com/rs/zerolog"
)

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
			Str(string(definitions.AlbumFieldID), a.ID)
		logging.Uint64If(e, "DBOID", a.DBOID)
	}
	if level <= zerolog.TraceLevel {
		e.Str(string(definitions.AlbumFieldDescription), a.Description).
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
	tplData.NavigationContext
	Album AlbumContext
}

type AlbumViewTree struct {
	Name     string
	ID       uint64
	ParentID *uint64
	Children []*AlbumViewTree
	EditUrl  template.URL
	ViewUrl  template.URL
	ACLLevel dbo.DBACLLevel
	Rank     uint64
}

func CreateAlbumView(album dbo.Album) *AlbumViewTree {
	return &AlbumViewTree{
		Name:     album.Name,
		ID:       *album.ID,
		ParentID: album.ParentID,
		Children: make([]*AlbumViewTree, 0),
		Rank:     album.Rank,
		ACLLevel: album.ACLLevel,
		EditUrl:  template.URL(routes.CreateAdminAlbumPath(*album.ID)),
		ViewUrl:  template.URL(routes.CreateAlbumPath(int64(*album.ID))),
	}
}

func (aw *AlbumViewTree) GetSorting() uint64 {
	return aw.Rank
}
func (aw *AlbumViewTree) GetID() uint64 {
	return aw.ID
}
func (aw *AlbumViewTree) GetParentID() *uint64 {
	return aw.ParentID
}
func (aw *AlbumViewTree) ClearChildren() {
	aw.Children = aw.Children[0:]
}
func (aw *AlbumViewTree) AddChild(child *AlbumViewTree) {
	aw.Children = append(aw.Children, child)
}
func (aw *AlbumViewTree) GetChildren() []*AlbumViewTree {
	return aw.Children
}

type AlbumsPageContext struct {
	tplData.NavigationContext
	Albums data.Forest[*AlbumViewTree]
}
