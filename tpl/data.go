package tpl

import (
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

type PageContext struct {
	Page   PageInfo
	Site   SiteInfo
	User   authData.ACLContext
	Images []*gridData.GridImage

	// szabadon bővíthető:
	Data any
}
type PageInfo struct {
	// melyik template kerüljön a <main>-be
	Role string
}
type SiteInfo struct {
	Title       string
	Author      string
	BaseURL     string
	Description string
	Footer      FooterInfo
	Logo        string
	Headline    string
}
type FooterInfo struct {
	Note          string
	CopyrightDate string
}
