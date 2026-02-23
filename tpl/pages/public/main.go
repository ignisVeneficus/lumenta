package public

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

		albumCtx := data.AlbumPageContex{}
		pageCtx := albumCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "landing", data.SurfacePublic)

		if err := r.RenderPage(c.Writer, "public/main", albumCtx); err != nil {
			c.String(500, err.Error())
		}
	}
}
