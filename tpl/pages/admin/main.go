package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func MainPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)

		pageCtx := &data.PageContext{}
		tpl.CreatePageContext(pageCtx, cfg, c, "main", data.SurfaceAdmin)

		if err := r.RenderPage(c.Writer, "admin/main", pageCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
