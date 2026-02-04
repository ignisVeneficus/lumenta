package tpl

import (
	"github.com/ignisVeneficus/lumenta/config"
	siteconfig "github.com/ignisVeneficus/lumenta/config/site"
)

func CreatePageContext(cfx config.Config) PageContext {
	ret := PageContext{
		Site: createSiteContext(cfx.Site),
	}

	return ret
}

func createSiteContext(cfx siteconfig.SiteConfig) SiteInfo {
	return SiteInfo{
		Title:       cfx.Title,
		Author:      cfx.Author,
		BaseURL:     cfx.BaseURL,
		Description: cfx.Description,
		Logo:        cfx.Logo,
		Headline:    cfx.Headline,
		Footer: FooterInfo{
			Note:          cfx.Footer.Note,
			CopyrightDate: cfx.Footer.CopyrightData,
		},
	}
}
