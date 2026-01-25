package template

import (
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/utils"
)

type TemplateResolver struct {
	roots   []string
	funcMap template.FuncMap
}

func NewTemplateResolver(roots ...string) *TemplateResolver {
	return &TemplateResolver{
		roots:   roots,
		funcMap: template.FuncMap{
			// később: asset, url, i18n stb
		},
	}
}

func (t *TemplateResolver) resolve(name string) (string, error) {
	for _, root := range t.roots {
		path := filepath.Join(root, name)
		ok, _ := utils.FileExists(path)
		if ok {
			return path, nil
		}
	}
	return "", fmt.Errorf("template not found: %s", name)
}

func Render(c *gin.Context, tpl string, data any) {
	/*
		resolver := TemplateResolver{}
		t, err := resolver.Load(tpl)
		if err != nil {
			c.String(500, err.Error())
			return
		}
		c.Status(200)
		_ = t.Execute(c.Writer, data)
	*/
}
