package derivative

import (
	"errors"
	"fmt"

	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (derivative DerivativesConfig) Validate(v *validate.ValidationErrors, path string) {
	if len(derivative) == 0 {
		err := errors.New("at least one derivative must be defined")
		validate.LogConfigError(path, nil, err)
		v.Add(err)
		return
	}

	seen := map[string]struct{}{}

	for i, d := range derivative {
		base := fmt.Sprintf("%s[%d]", path, i)
		name := d.validate(v, base)
		if name != "" {
			if _, ok := seen[name]; ok {
				err := errors.New("duplicate name")
				validate.LogConfigError(base+"/name", name, err)
				v.Add(err)
			}
			seen[name] = struct{}{}
		}
	}
}
func (d DerivativeConfig) validate(v *validate.ValidationErrors, path string) string {
	validate.RequireString(v, path+"/name", d.Name)

	switch {
	case (d.MaxWidth <= 0 || d.MaxHeight <= 0) && d.Mode == DerivativeSizeCrop:
		err := errors.New("invalid dimensions")
		validate.LogConfigError(path+"/size", map[string]int{
			"width":  d.MaxWidth,
			"height": d.MaxHeight,
		}, err)
		v.Add(err)
	case d.MaxWidth <= 0 && d.MaxHeight <= 0:
		err := errors.New("invalid dimensions")
		validate.LogConfigError(path+"/size", map[string]int{
			"width":  d.MaxWidth,
			"height": d.MaxHeight,
		}, err)
		v.Add(err)
	default:
		validate.LogConfigOK(path+"/size", map[string]int{
			"width":  d.MaxWidth,
			"height": d.MaxHeight,
		})
	}
	return d.Name
}
