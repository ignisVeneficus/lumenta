package tpl

import (
	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	siteconfig "github.com/ignisVeneficus/lumenta/config/site"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func CreatePageContext(pc *data.PageContext, cfg config.Config, c *gin.Context, role string, surface data.Surface) {
	pc.Site = createSiteContext(cfg.Site)
	pc.Page = createPageInfo(role, surface)
	pc.User = data.UserContext{
		ACLContext:      auth.GetAuthContex(c),
		HasGuestEnabled: cfg.Auth.GuestEnabled,
	}
}

func createSiteContext(cfx siteconfig.SiteConfig) data.SiteInfo {
	return data.SiteInfo{
		Title:       cfx.Title,
		Author:      cfx.Author,
		BaseURL:     cfx.BaseURL,
		Description: cfx.Description,
		Logo:        cfx.Logo,
		Headline:    cfx.Headline,
		Footer: data.FooterInfo{
			Note:          cfx.Footer.Note,
			CopyrightDate: cfx.Footer.CopyrightData,
		},
	}
}

func createPageInfo(role string, surface data.Surface) data.PageInfo {
	return data.PageInfo{
		PageRole: role,
		Surface:  surface,
	}
}
