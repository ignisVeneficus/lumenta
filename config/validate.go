package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog/log"
)

// Standardized error message helpers

func errRequired(field string) error {
	return fmt.Errorf("%s is required", field)
}

func errMin(field string, min any, value any) error {
	return fmt.Errorf("%s must be at least %v (got %v)", field, min, value)
}

func errOneOf(field string, allowed any, value any) error {
	return fmt.Errorf("%s must be one of %v (got %v)", field, allowed, value)
}

func requireString(v *ValidationErrors, path string, value string) bool {
	if strings.TrimSpace(value) == "" {
		err := errRequired(path)
		logConfigError(path, value, err)
		v.Add(err)
		return false
	}
	logConfigOK(path, value)
	return true
}

func requireIntMin(v *ValidationErrors, path string, value int, min int) bool {
	if value < min {
		err := errMin(path, min, value)
		logConfigError(path, value, err)
		v.Add(err)
		return false
	}
	logConfigOK(path, value)
	return true
}

func requireOneOf[T comparable](v *ValidationErrors, path string, value T, allowed []T) bool {
	for _, a := range allowed {
		if value == a {
			logConfigOK(path, value)
			return true
		}
	}
	err := errOneOf(path, allowed, value)
	logConfigError(path, value, err)
	v.Add(err)
	return false
}

func (c *Config) Validate() error {
	var verr ValidationErrors

	c.Server.validate(&verr, "server")
	c.Media.validate(&verr, "media")
	c.Gallery.validate(&verr, "gallery")
	c.Albums.validate(&verr, "albums")
	c.Auth.validate(&verr, "auth")
	c.Database.validate(&verr, "database")
	c.validateDerivatives(&verr, "derivatives")

	if verr.HasErrors() {
		return &verr
	}
	return nil
}

func (cfg *ServerConfig) validate(v *ValidationErrors, path string) {
	requireString(v, path+"/addr", cfg.Addr)

	checkDuration(v, path+"/timeouts/read", cfg.Timeouts.Read)
	checkDuration(v, path+"/timeouts/write", cfg.Timeouts.Write)
	checkDuration(v, path+"/timeouts/idle", cfg.Timeouts.Idle)
}

func (m *MediaConfig) validate(v *ValidationErrors, path string) {
	checkDir(path+"/originals", m.Originals, true, v)
	checkDir(path+"/derivatives", m.Derivatives, true, v)

	if m.Originals != "" &&
		m.Derivatives != "" &&
		m.Originals == m.Derivatives {

		err := errors.New("originals and derivatives must differ")
		logConfigError(path, map[string]string{
			"originals":   m.Originals,
			"derivatives": m.Derivatives,
		}, err)
		v.Add(err)
	}

}

func (g *GalleryConfig) validate(v *ValidationErrors, path string) {
	checkDir(
		path+"/templates/custom",
		g.Templates.Custom,
		false, // optional
		v,
	)
}

func (a *AlbumsConfig) validate(v *ValidationErrors, path string) {
	// TODO
	if a.Rebuild.BatchSize <= 0 {
		err := errors.New("batch_size must be > 0")
		logConfigError(path+"/rebuild/batch_size", a.Rebuild.BatchSize, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/rebuild/batch_size", a.Rebuild.BatchSize)
	}

	if a.Rebuild.Parallelism <= 0 {
		err := errors.New("parallelism must be > 0")
		logConfigError(path+"/rebuild/parallelism", a.Rebuild.Parallelism, err)
		v.Add(err)
	} else if a.Rebuild.Parallelism > 16 {
		err := errors.New("parallelism too high")
		logConfigError(path+"/rebuild/parallelism", a.Rebuild.Parallelism, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/rebuild/parallelism", a.Rebuild.Parallelism)
	}
}

func (a *AuthConfig) validate(v *ValidationErrors, path string) {
	switch a.Mode {
	case AuthModeForward:
		logConfigOK(path+"/mode", a.Mode)
		requireString(v, path+"/forward/user_header", a.Forward.UserHeader)

		if len(a.Forward.TrustedCIDRs) == 0 {
			err := errors.New("trusted_proxy_cidr must not be empty")
			logConfigError(path+"/forward/trusted_proxy_cidr", a.Forward.TrustedCIDRs, err)
			v.Add(err)
		} else {
			logConfigOK(path+"/forward/trusted_proxy_cidr", a.Forward.TrustedCIDRs)
		}
		if a.Forward.AdminRole == "" {
			err := errors.New("admin_role must be set")
			logConfigError(path+"/forward/admin_role", a.Forward.AdminRole, err)
			v.Add(err)
		} else {
			logConfigOK(path+"/forward/admin_role", a.Forward.AdminRole)
		}

	case AuthModeOIDC:
		logConfigOK(path+"/mode", a.Mode)

		if a.OIDC.Issuer == "" {
			err := errors.New("issuer must be set")
			logConfigError(path+"/oidc/issuer", a.OIDC.Issuer, err)
			v.Add(err)
		} else {
			logConfigOK(path+"/oidc/issuer", a.OIDC.Issuer)
		}

		if a.OIDC.ClientID == "" {
			err := errors.New("client_id must be set")
			logConfigError(path+"/oidc/client_id", a.OIDC.ClientID, err)
			v.Add(err)
		} else {
			logConfigOK(path+"/oidc/client_id", a.OIDC.ClientID)
		}
		if a.OIDC.AdminRole == "" {
			err := errors.New("admin_role must be set")
			logConfigError(path+"/oidc/admin_role", a.OIDC.AdminRole, err)
			v.Add(err)
		} else {
			logConfigOK(path+"/oidc/admin_role", a.OIDC.AdminRole)
		}

	default:
		err := errors.New("mode must be forward or oidc")
		logConfigError(path+"/mode", a.Mode, err)
		v.Add(err)
	}
}

func (c *Config) validateDerivatives(v *ValidationErrors, path string) {
	if len(c.Derivatives) == 0 {
		err := errors.New("at least one derivative must be defined")
		logConfigError(path, nil, err)
		v.Add(err)
		return
	}

	seen := map[string]struct{}{}

	for i, d := range c.Derivatives {
		base := fmt.Sprintf("%s[%d]", path, i)

		if d.Name == "" {
			err := errors.New("name missing")
			logConfigError(base+"/name", d.Name, err)
			v.Add(err)
		} else {
			logConfigOK(base+"/name", d.Name)
		}

		if _, ok := seen[d.Name]; ok {
			err := errors.New("duplicate name")
			logConfigError(base+"/name", d.Name, err)
			v.Add(err)
		}
		seen[d.Name] = struct{}{}

		switch {
		case (d.MaxWidth <= 0 || d.MaxHeight <= 0) && d.Mode == DerivateSizeCrop:
			err := errors.New("invalid dimensions")
			logConfigError(base+"/size", map[string]int{
				"width":  d.MaxWidth,
				"height": d.MaxHeight,
			}, err)
			v.Add(err)
		case d.MaxWidth <= 0 && d.MaxHeight <= 0:
			err := errors.New("invalid dimensions")
			logConfigError(base+"/size", map[string]int{
				"width":  d.MaxWidth,
				"height": d.MaxHeight,
			}, err)
			v.Add(err)
		default:
			logConfigOK(base+"/size", map[string]int{
				"width":  d.MaxWidth,
				"height": d.MaxHeight,
			})
		}
	}
}

func (d *DatabaseConfig) validate(v *ValidationErrors, path string) {
	if d.Host == "" {
		err := errors.New("host must be set")
		logConfigError(path+"/host", d.Host, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/host", d.Host)
	}

	if d.Port <= 0 || d.Port > 65535 {
		err := errors.New("invalid port")
		logConfigError(path+"/port", d.Port, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/port", d.Port)
	}

	if d.Name == "" {
		err := errors.New("database name must be set")
		logConfigError(path+"/name", d.Name, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/name", d.Name)
	}

	if d.User == "" {
		err := errors.New("user must be set")
		logConfigError(path+"/user", d.User, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/user", d.User)
	}

	if d.Password == "" {
		err := errors.New("password must be set")
		logConfigError(path+"/password", "***", err)
		v.Add(err)
	} else {
		logConfigOK(path+"/password", "***")
	}
}

type ValidationErrors struct {
	errors []error
}

func (v *ValidationErrors) Add(err error) {
	if err != nil {
		v.errors = append(v.errors, err)
	}
}

func (v *ValidationErrors) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *ValidationErrors) Error() string {
	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range v.errors {
		sb.WriteString(" - ")
		sb.WriteString(err.Error())
		sb.WriteRune('\n')
	}
	return sb.String()
}

func checkDuration(v *ValidationErrors, path string, d time.Duration) {
	if d <= 0 {
		err := errors.New("must be > 0")
		logConfigError(path, d, err)
		v.Add(fmt.Errorf("%s %w", path, err))
	} else {
		logConfigOK(path, d)
	}
}

func checkDir(pathKey string, dir string, required bool, v *ValidationErrors) {
	if dir == "" {
		if required {
			err := errors.New("directory must be set")
			logConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Info().
				Str("config", pathKey).
				Msg("directory not set (optional)")
		}
		return
	}

	info, err := os.Stat(dir)
	if err != nil {
		if required {
			logConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Warn().
				Str("config", pathKey).
				Str("value", dir).
				Err(err).
				Msg("optional directory does not exist")
		}
		return
	}

	if !info.IsDir() {
		err := errors.New("not a directory")
		if required {
			logConfigError(pathKey, dir, err)
			v.Add(fmt.Errorf("%s: %w", pathKey, err))
		} else {
			log.Warn().
				Str("config", pathKey).
				Str("value", dir).
				Msg("optional path exists but is not a directory")
		}
		return
	}

	// OK
	logConfigOK(pathKey, dir)
}

func (s *SyncConfig) validate(v *ValidationErrors, path string) {
	if len(s.Paths) == 0 {
		log.Logger.Info().
			Str("config", path+"/paths").
			Msg("no sync paths defined")
		return
	}

	for i := range s.Paths {
		s.Paths[i].validate(v, path, i)
	}
}

func (p *PathFilterConfig) validate(v *ValidationErrors, basePath string, idx int) {
	path := fmt.Sprintf("%s/paths[%d]", basePath, idx)

	if p.Path == "" {
		err := errors.New("path must be set")
		logConfigError(path+"/path", p.Path, err)
		v.Add(err)
	} else {
		logConfigOK(path+"/path", p.Path)
	}

	validateFilterGroup(&p.Filters, v, path+"/filters")
}

func validateFilterGroup(fg *ruleengine.FilterGroup, v *ValidationErrors, path string) {
	switch fg.Op {
	case ruleengine.OpAll, ruleengine.OpAny:
		logConfigOK(path+"/op", fg.Op)
	default:
		err := errors.New("invalid filter group op")
		logConfigError(path+"/op", fg.Op, err)
		v.Add(err)
	}

	if len(fg.Filters) == 0 {
		err := errors.New("filters must not be empty")
		logConfigError(path+"/filters", nil, err)
		v.Add(err)
		return
	}

	for i, f := range fg.Filters {
		if f == nil {
			err := errors.New("nil filter")
			logConfigError(fmt.Sprintf("%s/filters[%d]", path, i), nil, err)
			v.Add(err)
			continue
		}

		ft := f.FilterType()
		if ft == "" {
			err := errors.New("filter type missing")
			logConfigError(fmt.Sprintf("%s/filters[%d]/type", path, i), nil, err)
			v.Add(err)
			continue
		}

		logConfigOK(
			fmt.Sprintf("%s/filters[%d]/type", path, i),
			ft,
		)
	}
}

func (m *MetadataConfig) validate(v *ValidationErrors, path string) {
	if len(m.Fields) == 0 {
		log.Logger.Info().
			Str("config", path+"/metadata").
			Msg("no metadata fields defined")
		return
	}

	for key, field := range m.Fields {
		fieldPath := fmt.Sprintf("%s/metadata/fields/%s", path, key)

		if key == "" {
			err := errors.New("metadata field key must not be empty")
			logConfigError(fieldPath, nil, err)
			v.Add(err)
			continue
		}

		if len(field.Sources) == 0 {
			err := errors.New("no metadata sources defined")
			logConfigError(fieldPath+"/sources", nil, err)
			v.Add(err)
		}

		for i, src := range field.Sources {
			srcPath := fmt.Sprintf("%s/sources[%d]", fieldPath, i)

			if src.Ref == "" {
				err := errors.New("metadata source reference is empty")
				logConfigError(srcPath, nil, err)
				v.Add(err)
				continue
			}

			if !isValidMetadataRef(src.Ref) {
				err := errors.New("invalid metadata reference format (expected exif:*, iptc:*, xmp:*)")
				logConfigError(srcPath, src.Ref, err)
				v.Add(err)
			}
		}

		if field.Type != "" && !isValidMetaType(field.Type) {
			err := errors.New("invalid metadata type")
			logConfigError(fieldPath+"/type", field.Type, err)
			v.Add(err)
		}
	}
}

func isValidMetadataRef(ref string) bool {
	return strings.HasPrefix(ref, "exif:") ||
		strings.HasPrefix(ref, "iptc:") ||
		strings.HasPrefix(ref, "xmp:")
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

func (c *ExiftoolConfig) validate(v *ValidationErrors, path string) {
	if c.ResolvedPath == "" {
		err := errors.New("invalid exiftool path")
		logConfigError(path+"/path", c.Path, err)
		v.Add(err)
	}
}
