package tpl

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ignisVeneficus/lumenta/logging"
)

var (
	baseRoot = "web/templates"
)

type TemplateResolver struct {
	pages map[string]*template.Template
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

func NewTemplateResolver(ctx context.Context, userRoot string, funcMaps ...template.FuncMap) (*TemplateResolver, error) {
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

	// 3. funcmap merge
	mergedFuncMap := template.FuncMap{}
	for _, fm := range funcMaps {
		for k, v := range fm {
			mergedFuncMap[k] = v
		}
	}

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

	base := template.New("").Funcs(mergedFuncMap)

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

func (r *TemplateResolver) RenderPage(w io.Writer, page string, ctx any) error {

	tpl, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("page template not prepared: %s", page)
	}

	return tpl.ExecuteTemplate(w, "layout/baseof.html", ctx)
}
