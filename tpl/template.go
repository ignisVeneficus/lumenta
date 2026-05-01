package tpl

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/internal/locale"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

var (
	baseRoot    = "web/templates"
	iconMapFile = "/def/icons.yaml"
)

type TemplateResolver struct {
	pages map[string]*template.Template
}

func flattenMap(prefix string, input map[string]any, out map[string]string) {
	for k, v := range input {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			out[key] = val

		case map[string]any:
			flattenMap(key, val, out)

		default:

		}
	}
}

func loadYamlMapFile(ctx context.Context, file string) (data.IconMap, error) {
	logg := logging.Enter(ctx, "template.iconmap.load.yaml", map[string]any{"file": file})

	b, err := os.ReadFile(file)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(b, &data); err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	flat := map[string]string{}
	flattenMap("", data, flat)
	logging.Exit(logg, "ok", map[string]any{"found": len(flat)})
	return flat, nil
}

func loadIconMap(ctx context.Context, userRoot string) (data.IconMap, error) {
	logg := logging.Enter(ctx, "template.iconmap.load", map[string]any{"root": baseRoot, "user_root": userRoot})
	iconMap, err := loadYamlMapFile(ctx, baseRoot+iconMapFile)
	if err != nil {
		logging.ExitErr(logg, err)
		return iconMap, err
	}
	if userRoot != "" {
		usermap, err := loadYamlMapFile(ctx, userRoot+iconMapFile)
		if err != nil && !os.IsNotExist(err) {
			logging.ExitErr(logg, err)
			return nil, err
		}
		for k, v := range usermap {
			iconMap[k] = v
		}
	}
	logging.Exit(logg, "ok", map[string]any{"found": len(iconMap)})
	return iconMap, nil
}

func collectTemplates(ctx context.Context, root string) (map[string]string, error) {
	logg := logging.Enter(ctx, "template.collect", map[string]any{"root": root})
	result := map[string]string{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// rel path a kulcs (pl: partials/header.html)
		result[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		logging.ExitErr(logg, err)
	} else {
		logging.Exit(logg, "ok", map[string]any{"result": len(result)})
	}
	return result, err
}

func NewTemplateResolver(ctx context.Context, userRoot string, i18n *i18n.Service) (*TemplateResolver, error) {
	logg := logging.Enter(ctx, "template.resolver.create", map[string]any{"user_root": userRoot})
	allTpls, err := collectTemplates(ctx, baseRoot)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	if userRoot != "" {
		userTpls, err := collectTemplates(ctx, userRoot)
		if err != nil && !os.IsNotExist(err) {
			logging.ExitErr(logg, err)
			return nil, err
		}

		for k, v := range userTpls {
			allTpls[k] = v
		}
	}

	iconMap, err := loadIconMap(ctx, userRoot)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}

	funcMap := DefaultFuncMap(i18n, iconMap)

	baseTpls := map[string]string{}
	pageTpls := map[string]string{}

	for name, content := range allTpls {
		if strings.HasPrefix(name, "pages/") {
			page := strings.TrimSuffix(
				strings.TrimPrefix(name, "pages/"),
				".html",
			)
			pageTpls[page] = content
		} else {
			baseTpls[name] = content
		}
	}

	base := template.New("").Funcs(funcMap)

	for name, content := range baseTpls {
		if _, err := base.New(name).Parse(content); err != nil {
			err = fmt.Errorf("base template %s: %w", name, err)
			logging.ExitErr(logg, err)
			return nil, err
		}
	}

	pages := map[string]*template.Template{}
	for page, content := range pageTpls {
		tpl, err := base.Clone()
		if err != nil {
			return nil, err
		}

		if _, err := tpl.New("pages/" + page + ".html").Parse(content); err != nil {
			return nil, fmt.Errorf("page %s: %w", page, err)
		}

		pages[page] = tpl
	}

	logging.Exit(logg, "ok", nil)
	return &TemplateResolver{
		pages: pages,
	}, nil
}

func (r *TemplateResolver) RenderPage(w io.Writer, page string, ctx any, loc string, i18n *i18n.Service) error {

	tpl, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("page template not prepared: %s", page)
	}

	t, err := tpl.Clone()
	if err != nil {
		return err
	}
	// override, enclose the actual language into the func map
	t = t.Funcs(template.FuncMap{
		"formatTime": func(v time.Time) string {
			return locale.FormatTime(v, loc)
		},
		"formatDuration": func(d time.Duration) string {
			return locale.FormatDuration(&d, loc, i18n)
		},
		"formatNumber": func(n any) string {
			return locale.FormatNumber(n, loc)
		},
		"t": func(key string, args ...any) string {
			var params map[string]any
			if len(args) > 0 {
				params = args[0].(map[string]any)
			}
			return i18n.T(loc, key, params)
		},
		"i18n_js": func(selectors ...string) template.JS {
			m := i18n.ExtractKeys(loc, selectors)
			b, _ := json.Marshal(m)
			return template.JS(b)
		},
	})

	return t.ExecuteTemplate(w, "layout/baseof.html", ctx)
}
