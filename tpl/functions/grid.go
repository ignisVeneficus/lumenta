package functions

import (
	"fmt"
	"html/template"
	"strings"

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
