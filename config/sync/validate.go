package sync

import (
	"errors"
	"fmt"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config/validate"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog/log"
)

func (s *SyncConfig) Validate(v *validate.ValidationErrors, path string) {
	if len(s.Paths) == 0 {
		log.Logger.Info().
			Str("config", path+"/paths").
			Msg("no sync paths defined")
	}

	for i := range s.Paths {
		s.Paths[i].validate(v, path, i)
	}

	if len(s.Extensions) == 0 {
		err := validate.ErrRequired(path + "/extensions")
		validate.LogConfigError(path+"/extensions", nil, err)
		v.Add(err)
	}
	s.Metadata.validate(v, path+"/metadata")
	s.Exiftool.validate(v, path+"/exiftool")
	if s.Panorama != nil {
		validateFilterGroup(s.Panorama, v, path+"/panorama")
	}

}

func (p *PathFilterConfig) validate(v *validate.ValidationErrors, basePath string, idx int) {
	path := fmt.Sprintf("%s/paths[%d]", basePath, idx)
	validate.RequireString(v, path+"/root", p.Root)

	validateFilterGroup(&p.Filters, v, path+"/filters")
}

func validateFilterGroup(fg *ruleengine.RuleGroup, v *validate.ValidationErrors, path string) {
	switch fg.Op {
	case ruleengine.OpAll, ruleengine.OpAny:
		validate.LogConfigOK(path+"/op", fg.Op)
	default:
		err := errors.New("invalid filter group op")
		validate.LogConfigError(path+"/op", fg.Op, err)
		v.Add(err)
	}

	if len(fg.Rules) == 0 {
		err := validate.ErrRequired(path + "/filters")
		validate.LogConfigError(path+"/filters", nil, err)
		v.Add(err)
		return
	}

	for i, f := range fg.Rules {
		if f == nil {
			err := errors.New("nil filter")
			validate.LogConfigError(fmt.Sprintf("%s/filters[%d]", path, i), nil, err)
			v.Add(err)
			continue
		}

		validate.RequireString(v, fmt.Sprintf("%s/filters[%d]/type", path, i), f.FilterType())
	}
}

func (m *MetadataConfig) validate(v *validate.ValidationErrors, path string) {
	if len(m.Fields) == 0 {
		log.Logger.Info().
			Str("config", path+"/fields").
			Msg("no metadata fields defined")
	}
	for key, field := range m.Fields {
		fieldPath := fmt.Sprintf("%s/fields/%s", path, key)

		validate.RequireString(v, fieldPath+" key", key)

		if len(field.Sources) == 0 {
			err := validate.ErrRequired(path + "/sources")
			validate.LogConfigError(fieldPath+"/sources", nil, err)
			v.Add(err)
		}

		for i, src := range field.Sources {
			srcPath := fmt.Sprintf("%s/sources[%d]", fieldPath, i)

			validate.RequireString(v, srcPath+"/ref", src.Ref)
		}
		if field.Type != "" && !isValidMetaType(field.Type) {
			err := errors.New("invalid metadata type")
			validate.LogConfigError(fieldPath+"/type", field.Type, err)
			v.Add(err)
		}
	}
}

func isValidMetaType(t data.MetadataType) bool {
	switch t {
	case data.MetaString,
		data.MetaInt,
		data.MetaFloat,
		data.MetaBool,
		data.MetaRational,
		data.MetaList,
		data.MetaDateTime:
		return true
	default:
		return false
	}
}

func (c *ExiftoolConfig) validate(v *validate.ValidationErrors, path string) {
	if c.ResolvedPath == "" {
		err := errors.New("invalid exiftool path")
		validate.LogConfigError(path+"/path", c.Path, err)
		v.Add(err)
	}
}

func (ac *ACLRules) validate(v *validate.ValidationErrors, path string) {
	for i, r := range *ac {
		r.validate(v, path, i)
	}
}
func (acr *ACLRule) validate(v *validate.ValidationErrors, basePath string, idx int) {
	path := fmt.Sprintf("%s/[%d]", basePath, idx)
	validate.RequireString(v, path+"/role", string(acr.Role))
	if !authData.IsValidRole(acr.Role) {
		err := fmt.Errorf("meta_acl: invalid role level")
		validate.LogConfigError(path+"/role", acr.Role, err)
		v.Add(err)
	}
	for i, rule := range acr.Rules {
		rulePath := fmt.Sprintf("%s/rule[%d]", path, i)
		validateFilterGroup(rule, v, rulePath)
	}

}
