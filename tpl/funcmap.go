package tpl

import (
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl/functions"
)

func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"icon":          Icon,
		"siteIcon":      SiteIcon,
		"tileImgList":   functions.TileImgList,
		"gridRectToVar": functions.RectsToStyleVars,
		"gridTileRole":  functions.TileRole,
		"toPercent":     ToPercent,

		"pagingFirst": functions.PagingFirst,
		"pagingPrev":  functions.PagingPrev,
		"pagingNext":  functions.PagingNext,
		"pagingLast":  functions.PagingLast,

		"imagePath": ImagePath,
		"tagsRootPath": func() template.URL {
			return template.URL(routes.CreateTagsRootPath())
		},
		"tagPath": TagPath,

		"adminRootPath": func() template.URL {
			return template.URL(routes.CreateAdminRootPath())
		},

		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"reticleOffset": functions.FocusOffset,

		"formatLatitude":  FormatLatDMS,
		"formatLongitude": FormatLonDMS,
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
func TagPath(tagId uint64) template.URL {
	return template.URL(routes.CreateTagPath(uint64(tagId)))
}

func FormatLatDMS(lat float64) string {
	return toDMS(lat, "N", "S")
}

func FormatLonDMS(lon float64) string {
	return toDMS(lon, "E", "W")
}
func toDMS(decimal float64, posHem, negHem string) string {
	abs := math.Abs(decimal)

	deg := int(abs)
	minFloat := (abs - float64(deg)) * 60
	min := int(minFloat)
	sec := (minFloat - float64(min)) * 60

	// rounding guard
	if sec >= 59.9995 {
		sec = 0
		min++
	}
	if min >= 60 {
		min = 0
		deg++
	}

	hem := posHem
	if decimal < 0 {
		hem = negHem
	}

	return fmt.Sprintf("%d°%02d'%05.2f\"%s", deg, min, sec, hem)
}
