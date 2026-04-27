package public

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/internal/i18n"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func TagsRootPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		logg := logging.Enter(c, "page.public.tags.root", nil)

		breadCrumbs := data.Breadcrumbs{
			data.Breadcrumb{
				Label: "Tags",
			},
		}
		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		tags, err := dao.QueryTagsByACL(database, c, acl.ACLContext)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		tagTree := dbo.BuildTagsWCountTree(dbo.TagsWCountToPointer(tags))

		trCtx := data.TagRootPageContext{}
		pageCtx := trCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)
		trCtx.Breadcrumbs = breadCrumbs
		trCtx.TagsTree = tagTree

		logging.Exit(logg, "ok", nil)

		if err := r.RenderPage(c.Writer, "public/tags-root", trCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logg, err)
			return
		}
		logging.Exit(logg, "ok", nil)
	}
}
