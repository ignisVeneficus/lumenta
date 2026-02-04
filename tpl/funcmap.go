package tpl

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/ignisVeneficus/lumenta/tpl/functions"
)

func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// debug
		"dump": func(v any) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"icon":            Icon,
		"siteIcon":        SiteIcon,
		"gridTileImgList": TileImgList,
		"gridTileImg":     TileImg,
		"toPercent":       ToPercent,
		"gridRectToVar":   functions.RectsToStyleVars,
		"gridTileRole":    functions.TileRole,

		// később:
		// "asset": AssetFunc,
		// "url":   URLFunc,
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

var ImgSrc = "/img/%d/%s"

func genImgSrc(id uint64, derivate string) string {
	return fmt.Sprintf(ImgSrc, id, derivate)
}

func genImgSrcToList(id uint64, width int) string {
	derivate := fmt.Sprintf("w%d", width)
	return fmt.Sprintf("%s %dw", genImgSrc(id, derivate), width)
}
func TileImgList(imgID uint64, widths ...int) template.Srcset {
	var parts []string
	for _, w := range widths {
		parts = append(parts, genImgSrcToList(imgID, w))
	}
	return template.Srcset(strings.Join(parts, ", "))
}
func TileImg(imgID uint64) template.HTML {
	return template.HTML(genImgSrc(imgID, "w1024"))
}
