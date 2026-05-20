package tpl

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ignisVeneficus/lumenta/definitions"
	i18n "github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/internal/locale"
	"github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/tpl/functions"
)

func DefaultFuncMap(i18n *i18n.Service, iconMap data.IconMap) template.FuncMap {
	return template.FuncMap{
		"svgIcon":       Icon,
		"siteIcon":      SiteIcon,
		"tileImgList":   functions.TileImgList,
		"gridRectToVar": functions.RectsToStyleVars,
		"gridTileRole":  functions.TileRole,
		"toPercent":     ToPercent,

		"imagePath":      functions.ImagePath,
		"tagsRootPath":   functions.TagsRootPath,
		"tagPath":        functions.TagPath,
		"albumsRootPath": functions.AlbumsRootPath,

		"adminRootPath":     functions.AdminRootPath,
		"adminImagePath":    functions.AdminImagePath,
		"adminAlbumsPath":   functions.AdminAlbumsPath,
		"adminAlbumPath":    functions.AdminAlbumPath,
		"adminAlbumNewPath": functions.AdminAlbumNewPath,
		"adminFsPath":       functions.AdminFsPath,
		"adminSyncRunsPath": functions.AdminSyncRunsPath,
		"adminSyncRunPath":  functions.AdminSyncRunPath,

		"adminSyncFilesPathPath": functions.AdminSyncFilesPathPath,
		"adminSyncFilesPath":     functions.AdminSyncFilesPath,
		"adminSyncFilePath":      functions.AdminSyncFilePath,

		"apiAdminAlbumPathJS": functions.ApiAdminAlbumPathJS,
		"apiAdminAlbumsPath":  functions.ApiAdminAlbumsPathView,
		"apiAdminImagePath":   functions.ApiAdminImagePath,
		"apiAdminTagsPath":    functions.ApiAdminTagsPathView,

		"reticleOffset": functions.FocusOffset,

		"formatLatitude":  FormatLatDMS,
		"formatLongitude": FormatLonDMS,

		"toJS": func(v string) template.JS {
			return template.JS(v)
		},
		"toJSON": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},

		"fieldName": func(fn string) definitions.FieldName {
			return definitions.FieldName(fn)
		},

		"warpPath": CreateSpacePath,

		"dict": func(values ...any) map[string]any {
			m := make(map[string]any)
			for i := 0; i < len(values); i += 2 {
				m[values[i].(string)] = values[i+1]
			}
			return m
		},
		"i": func(key, title string, classes ...string) data.IconSpec {
			ic := make([]string, 0, len(classes))
			for _, c := range classes {
				if c != "" {
					ic = append(ic, c)
				}
			}
			return data.IconSpec{
				Key:   key,
				Title: title,
				Class: strings.Join(ic, " "),
			}
		},
		"iconLookup": func(key, failback string) string {
			v, ok := iconMap[key]
			if !ok {
				return failback
			}
			return v
		},

		// placeholderek
		"formatTime": func(v time.Time) string {
			return locale.FormatTime(v, "_")
		},
		"formatDuration": func(d time.Duration) string {
			return locale.FormatDuration(&d, "_", i18n)
		},
		"formatNumber": func(n any) string {
			return locale.FormatNumber(n, "_")
		},
		"t": func(key string, args ...any) string {
			var params map[string]any
			if len(args) > 0 {
				params = args[0].(map[string]any)
			}
			return i18n.T("_", key, params)
		},
		"i18n_js": func(selectors ...string) template.JS {
			m := i18n.ExtractKeys("_", selectors)
			b, _ := json.Marshal(m)
			return template.JS(b)
		},
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
