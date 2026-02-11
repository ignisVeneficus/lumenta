package tpl

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl/functions"
)

func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"icon":            Icon,
		"siteIcon":        SiteIcon,
		"gridTileImgList": functions.TileImgList,
		"gridRectToVar":   functions.RectsToStyleVars,
		"gridTileRole":    functions.TileRole,
		"toPercent":       ToPercent,

		"linkAdmin": func() template.URL {
			return template.URL(routes.CreateAdminRootPath())
		},
		"pagingFirst": functions.PagingFirst,
		"pagingPrev":  functions.PagingPrev,
		"pagingNext":  functions.PagingNext,
		"pagingLast":  functions.PagingLast,

		"imagePath": ImagePath,
	}
}

// functions

func ToPercent(v float32) template.HTML {
	val := fmt.Sprintf("%.2f%%", v*100.0)
	return template.HTML(val)
}

const iconBasePath = "web/assets/icons"
const siteIconBasePath = "web/static"

// Icon renders an inline SVG icon.
// name  – icon filename without .svg
// class – CSS classes added to <svg>
// aria  – aria-label text; if empty, aria-hidden="true" is used
func Icon(name, class, aria string) template.HTML {
	path := filepath.Join(iconBasePath, name+".svg")
	begin := "<span"
	data, err := os.ReadFile(path)
	if err != nil {
		// Fail silently: broken icon should not break the page
		return ""
	}

	svg := string(data)
	if class != "" {
		begin += ` class="` + template.HTMLEscapeString(class) + `" `
	}
	if aria != "" {
		begin += ` role="img" aria-label="` + template.HTMLEscapeString(aria) + `" `
	} else {
		begin += ` aria-hidden="true" `
	}
	begin += ">"
	svg = begin + svg + "</span>"

	return template.HTML(svg)
}

// SiteIcon renders an inline SVG icon.
// path  – icon pathname under /web/static/
// class – CSS classes added to <svg>
// aria  – aria-label text; if empty, aria-hidden="true" is used
func SiteIcon(path string) template.HTML {
	path = filepath.Join(siteIconBasePath, path)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	svg := string(data)
	return template.HTML(svg)
}

func ImagePath(imageId uint64, derivative string) template.URL {
	return template.URL(routes.CreateDerivativePath(imageId, derivative))
}
