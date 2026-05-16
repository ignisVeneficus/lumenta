package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"time"

	focusdata "github.com/ignisVeneficus/lumenta/data"
	rootData "github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	grid "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

type ImagePageContext struct {
	data.NavigationContext
	Image PageImage
	Users []dbo.User
}

type ImageSync struct {
	LastSync    time.Time
	LastStatus  string
	LastUpdated *time.Time
}

type PageImage struct {
	dbo.Image
	Realpath      string
	ComputedFocus focusdata.Focus
	Aspect        grid.Aspect
	Sync          ImageSync
	SingleMap     *data.SingleMap
	Form          ImageForm
	Covers        []routes.AlbumID
	Tags          rootData.Forest[*data.ViewTreeNode]
}

func (pi PageImage) CoversArray() template.JS {
	if pi.Covers == nil {
		return template.JS("[]")
	}
	result := make([]string, len(pi.Covers))
	for i, v := range pi.Covers {
		result[i] = strconv.FormatUint(uint64(v), 10)
	}
	b, _ := json.Marshal(result)
	return template.JS(b)
}

type ImageForm struct {
	ACLLevel  string `form:"acl_scope"`
	ACLUserID string `form:"acl_user"`
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
	return pi.Caption != nil && (*pi.Caption) != ""
}
func (pi PageImage) NormalisedSubject() string {
	if pi.HasSubject() {
		return *pi.Caption
	}
	return EMDash
}
func (pi PageImage) RoutesImagedID() routes.ImageID {
	return routes.ImageID(*pi.ID)
}
