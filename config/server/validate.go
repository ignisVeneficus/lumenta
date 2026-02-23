package server

import "github.com/ignisVeneficus/lumenta/config/validate"

func (cfg *ServerConfig) Validate(v *validate.ValidationErrors, path string) {
	validate.RequireString(v, path+"/addr", cfg.Addr)

	validate.CheckDuration(v, path+"/timeouts/read", cfg.Timeouts.Read)
	validate.CheckDuration(v, path+"/timeouts/readHeader", cfg.Timeouts.Header)
	validate.CheckDuration(v, path+"/timeouts/write", cfg.Timeouts.Write)
	validate.CheckDuration(v, path+"/timeouts/idle", cfg.Timeouts.Idle)
}
