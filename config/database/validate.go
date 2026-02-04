package database

import (
	"errors"

	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (d DatabaseConfig) Validate(v *validate.ValidationErrors, path string) {
	validate.RequireString(v, path+"/host", d.Host)

	if d.Port <= 0 || d.Port > 65535 {
		err := errors.New("invalid port")
		validate.LogConfigError(path+"/port", d.Port, err)
		v.Add(err)
	} else {
		validate.LogConfigOK(path+"/port", d.Port)
	}

	validate.RequireString(v, path+"/name", d.Name)
	validate.RequireString(v, path+"/user", d.User)

	if d.Password == "" {
		err := errors.New("password must be set")
		validate.LogConfigError(path+"/password", "***", err)
		v.Add(err)
	} else {
		validate.LogConfigOK(path+"/password", "***")
	}
}
