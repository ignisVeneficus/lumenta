package presentation

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/db/dbo"

	"github.com/ignisVeneficus/lumenta/config/validate"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

func (p *PresentationConfig) Validate(v *validate.ValidationErrors, path string) {
	validate.CheckDir(path+"/templates", p.Templates, false, v)
	p.Grid.validate(v, path+"/grid")
	p.MetadataACL.validate(v, path+"/metadata_acl")
	if p.TagMeaningConfig != nil {
		p.TagMeaningConfig.validate(v, path+"/tag_meaning")
	}
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
func (m MetadataACLConfig) validate(v *validate.ValidationErrors, path string) {
	for role := range m {
		if !dbo.IsValidRole(role) {
			err := fmt.Errorf("meta_acl: invalid role level")
			validate.LogConfigError(path, role, err)
			v.Add(err)
		}
	}

}
func (tmc TagMeaningConfig) validate(v *validate.ValidationErrors, path string) {
	if tmc.Threshold == 0 {
		err := fmt.Errorf("threshold: must set")
		validate.LogConfigError(path, nil, err)
		v.Add(err)
	}
	tmc.MeaningMap.validate(v, path+"/map")
}
func (tm TagMeaningMap) validate(v *validate.ValidationErrors, path string) {
	if len(tm) == 0 {
		err := fmt.Errorf("must set")
		validate.LogConfigError(path, nil, err)
		v.Add(err)
		return
	}
	for k, tag := range tm {
		if !IsValidTagMeaning((k)) {
			err := fmt.Errorf("key: invalid meaning")
			validate.LogConfigError(path, k, err)
			v.Add(err)
		}
		p := path + "/" + string(k) + "/"
		if len(tag.TagRoots) == 0 {
			err := fmt.Errorf("value: empty list")
			validate.LogConfigError(p+"roots", nil, err)
			v.Add(err)
		}
		for i, f := range tag.Features {
			if !IsValidTagFeature(f) {
				err := fmt.Errorf("value: empty list")
				validate.LogConfigError(fmt.Sprintf("%sfeature[%d]", p, i), nil, err)
				v.Add(err)
			}
		}
	}
}
