package filesystem

import (
	"fmt"
	"path"

	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (m FilesystemConfig) Validate(v *validate.ValidationErrors, confPath string) {
	validate.CheckDir(confPath+"/derivatives", m.Derivatives, true, v)

	if len(m.Originals) == 0 {
		err := fmt.Errorf("at least one originals must set")
		validate.LogConfigError(confPath+"/originals", "", err)
		v.Add(err)
	}
	for name, conf := range m.Originals {
		validate.CheckDir(fmt.Sprintf("%s/originals[%s]/root", confPath, name), conf.Root, true, v)
		for i, exclude := range conf.Excluded {
			exPath := path.Join(conf.Root, exclude)
			validate.CheckDir(fmt.Sprintf("%s/originals[%s]/excluded[%d]", confPath, name, i), exPath, false, v)
		}
	}
}
