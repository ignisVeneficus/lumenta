package pages

import (
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/grid"
)

func MainPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)
		// TODO: just for testing, need rewrite the whole
		data := tpl.CreatePageContext(cfg)
		data.Page = tpl.PageInfo{
			Role: "landing",
		}
		/*
			Site: SiteInfo{
				Title:       "Lumenta",
				Author:      "Ignis",
				Description: "kep oldal",
				Date:        "2026",
				Logo:        "img/lumenta.svg",
				Headline:    "Just photos. Properly organized.",
				Footer: FooterInfo{
					Note: "I touch nothing here · Only space learns where to rest · Structure completes itself",
				},
			},
		*/
		data.User = auth.GetAuthContex(c)
		imgIds := []int{522}
		for id := 535; id < 559; id++ {
			imgIds = append(imgIds, id)
		}
		database := db.GetDatabase()

		images := []dbo.Image{}
		for _, id := range imgIds {
			img, err := dao.GetImageById(database, c, uint64(id))
			if err != nil {
				log.Logger.Error().Err(err).Msg("sh*t happens")
			} else {
				images = append(images, img)
			}
		}
		gridImages := grid.BuildGrid(images, cfg.Presentation.Grid, 200)
		data.Images = gridImages

		if err := r.RenderPage(c.Writer, "main", data); err != nil {
			c.String(500, err.Error())
		}
	}
}
