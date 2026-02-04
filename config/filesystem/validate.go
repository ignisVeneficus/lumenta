package filesystem

import (
	"errors"

	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (m FilesystemConfig) Validate(v *validate.ValidationErrors, path string) {
	validate.CheckDir(path+"/originals", m.Originals, true, v)
	validate.CheckDir(path+"/derivatives", m.Derivatives, true, v)

	if m.Originals != "" &&
		m.Derivatives != "" &&
		m.Originals == m.Derivatives {

		err := errors.New("originals and derivatives must differ")
		validate.LogConfigError(path, map[string]string{
			"originals":   m.Originals,
			"derivatives": m.Derivatives,
		}, err)
		v.Add(err)
	}

}
