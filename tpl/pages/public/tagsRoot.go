package public

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
)

func TagsRootPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/public/tag/root", nil, nil)

		breadCrumbs := tplData.Breadcrumbs{
			tplData.Breadcrumb{
				Label: "Tags",
			},
		}
		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		tags, err := dao.QueryTagsByACL(database, ctx, acl.ACLContext)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		flatForrest := tplData.NewFlatForrest()
		tagsList := tplData.MapToViewNodes(tags,
			func(t dbo.TagWCount) tplData.ViewTreeNode {
				return tplData.ViewTreeNode{
					ID:       *t.ID,
					ParentID: t.ParentID,
					Label:    t.Name,
					Notes: []string{
						i18n.T(loc, "page.public.tags.nr_images", map[string]any{
							"images": t.Count,
						}),
					},
					URL: template.URL(routes.CreateTagPath(*t.ID)),
				}
			})
		flatForrest.Add(tagsList)
		forest := flatForrest.Build()
		tplData.SetTagsMeaning(forest, cfg.Presentation.TagMeaningConfig)

		trCtx := tplData.TagRootPageContext{}
		pageCtx := trCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", tplData.SurfacePublic)
		trCtx.Breadcrumbs = breadCrumbs
		trCtx.TagsTree = *forest

		logging.Exit(logScope, "ok", nil)

		if err := r.RenderPage(c.Writer, "public/tags-root", trCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
