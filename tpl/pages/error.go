package pages

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func Global404(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		errorCtx := data.ErrorPageContext{}

		pageCtx := errorCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "error", data.SurfacePublic)
		errorCtx.PageContext = *pageCtx
		errorCtx.Resource = "Page"
		errorCtx.RootURL = "/"

		errorCtx.Title = "Page not found"
		errorCtx.Message = "The requested page does not exist or may have been removed."
		errorCtx.BackLabel = "Back to the website"

		c.Status(http.StatusNotFound)

		r.RenderPage(c.Writer, "404", errorCtx)
	}
}

func Soft404(r *tpl.TemplateResolver, cfg config.Config, c *gin.Context,
	surface data.Surface, resource, rootURL string, resourceID uint64) {

	errorCtx := data.ErrorPageContext{}

	pageCtx := errorCtx.GetPage()
	tpl.CreatePageContext(pageCtx, cfg, c, "error", surface)

	errorCtx.PageContext = *pageCtx

	errorCtx.Resource = resource
	errorCtx.ResourceID = resourceID
	errorCtx.RootURL = rootURL

	errorCtx.Title = fmt.Sprintf("%s not found", resource)
	errorCtx.Message = fmt.Sprintf(
		"The requested %s does not exist or may have been removed.",
		strings.ToLower(resource),
	)
	errorCtx.BackLabel = fmt.Sprintf("Back to %ss", strings.ToLower(resource))

	c.Status(http.StatusNotFound)

	r.RenderPage(c.Writer, "404", errorCtx)

	c.Abort()
}
