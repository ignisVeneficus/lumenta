package presentation

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/config/validate"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

func (p *PresentationConfig) Validate(v *validate.ValidationErrors, path string) {
	validate.CheckDir(path+"/templates", p.Templates, false, v)
	p.Grid.validate(v, path+"/grid")
}

func validateSpan(s gridData.Span, v *validate.ValidationErrors, path string) {
	validate.RequireIntMin(v, path+"/w", s.W, 1)
	validate.RequireIntMin(v, path+"/h", s.H, 1)
}

func (a AspectConfig) validate(v *validate.ValidationErrors, path string) {
	requiredAspects := []string{
		string(gridData.AspectNormal),
		string(gridData.AspectLandscape),
		string(gridData.AspectPanorama),
		string(gridData.AspectTall),
	}

	seen := make(map[string]bool)

	for aspect, span := range a {
		aspectPath := path + "/" + aspect

		if validate.RequireOneOf(v, aspectPath, aspect, requiredAspects) {
			seen[aspect] = true
		}
		validateSpan(span, v, aspectPath)
	}

	for _, req := range requiredAspects {
		if !seen[req] {
			err := validate.ErrRequired(path + "/" + req)
			validate.LogConfigError(path+"/"+req, nil, err)
			v.Add(err)
		}
	}
}

func (g RoleConfig) validate(v *validate.ValidationErrors, path string) {
	requiredRoles := []string{
		string(gridData.RoleNormal),
		string(gridData.RoleLarge),
		string(gridData.RoleHero),
	}

	seen := make(map[string]bool)

	for role, aspects := range g {
		rolePath := path + "/" + role

		if validate.RequireOneOf(v, rolePath, role, requiredRoles) {
			seen[role] = true
		}
		aspects.validate(v, rolePath)
	}

	// hiányzó role-ok
	for _, req := range requiredRoles {
		if !seen[req] {
			err := validate.ErrRequired(path + "/" + req)
			validate.LogConfigError(path+"/"+req, nil, err)
			v.Add(err)
		}
	}
}

func (g GridConfig) validate(v *validate.ValidationErrors, path string) {
	if len(g) == 0 {
		err := validate.ErrRequired(path)
		validate.LogConfigError(path, g, err)
		v.Add(err)
		return
	}

	for gridWidth, roles := range g {
		gridPath := fmt.Sprintf("%s/%d", path, gridWidth)

		validate.RequireIntMin(v, gridPath, gridWidth, 1)
		roles.validate(v, gridPath)
	}
}
