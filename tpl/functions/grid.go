package functions

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/ignisVeneficus/lumenta/server/routes"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

func RectsToStyleVars(img gridData.GridImage) template.HTMLAttr {
	var b strings.Builder

	for grid, r := range img.Layouts {
		fmt.Fprintf(&b,
			"--p%dx:%d;--p%dy:%d;--p%dw:%d;--p%dh:%d;",
			grid, r.Rect.X,
			grid, r.Rect.Y,
			grid, r.Rect.W,
			grid, r.Rect.H,
		)
		fmt.Fprintf(&b,
			"--p%dminx:%.2f%%;--p%dmaxx:%.2f%%;--p%dminy:%.2f%%;--p%dmaxy:%.2f%%;",
			grid, r.Clamp.MinX*100.0,
			grid, r.Clamp.MaxX*100.0,
			grid, r.Clamp.MinY*100.0,
			grid, r.Clamp.MaxY*100.0,
		)
	}
	return template.HTMLAttr(`style="` + b.String() + `"`)
}
func TileRole(img gridData.GridImage) string {
	ret := []string{}
	for gridW, l := range img.Layouts {
		ret = append(ret, fmt.Sprintf("role%d-%s", gridW, l.Role))
	}
	return strings.Join(ret, " ")

}

func genImgSrc(id uint64, width int) string {
	derivative := fmt.Sprintf("w%d", width)
	return fmt.Sprintf("%s %dw", routes.CreateDerivativePath(id, derivative), width)
}
func TileImgList(imgID uint64, widths ...int) template.Srcset {
	var parts []string
	for _, w := range widths {
		parts = append(parts, genImgSrc(imgID, w))
	}
	return template.Srcset(strings.Join(parts, ", "))
}

func TileImg(imgID uint64, width int) template.HTML {
	return template.HTML(genImgSrc(imgID, width))
}
