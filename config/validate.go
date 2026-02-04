package config

import (
	"github.com/ignisVeneficus/lumenta/config/validate"
)

func (c *Config) Validate() error {
	var verr validate.ValidationErrors

	c.Server.Validate(&verr, "server")
	c.Filesystem.Validate(&verr, "filesystem")
	c.Auth.Validate(&verr, "auth")
	c.Database.Validate(&verr, "database")
	c.Derivatives.Validate(&verr, "derivatives")
	c.Sync.Validate(&verr, "sync")
	c.Site.Validate(&verr, "site")
	c.Presentation.Validate(&verr, "sync")

	if verr.HasErrors() {
		return &verr
	}
	return nil
}
