package public

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

/* TODO: create global func */

func TagsRootPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		logg := logging.Enter(c, "public.tags.root", nil)

		breadCrumbs := data.Breadcrumbs{
			data.Breadcrumb{
				Label: "Tags",
			},
		}
		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)
		tags, err := dao.QueryTagsByACL(database, c, tpl.CreateDBOACL(acl))
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		tagTree := dbo.BuildTagsWCountTree(tags)

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)

		trCtx := data.TagRootPageContext{}
		pageCtx := trCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "tags", data.SurfacePublic)
		trCtx.Breadcrumbs = breadCrumbs
		trCtx.TagsTree = tagTree

		if err := r.RenderPage(c.Writer, "public/tags-root", trCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
