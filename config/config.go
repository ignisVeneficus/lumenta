package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	ENV_PREFIX = "LUMENTA"
)

type AuthMode string
type DerivateSizeMode string

var (
	LogConfigEnv                        = ENV_PREFIX + "_LOG_CONFIG"
	AuthModeForward    AuthMode         = "forward"
	AuthModeOIDC       AuthMode         = "oidc"
	DerivateSizeCrop   DerivateSizeMode = "crop"
	DerivateSizeResize DerivateSizeMode = "fit"
)

type Config struct {
	Env         Environment        `yaml:"-"` // kizárólag ENV-ből
	Server      ServerConfig       `yaml:"server"`
	Database    DatabaseConfig     `yaml:"database"`
	Media       MediaConfig        `yaml:"media"`
	Gallery     GalleryConfig      `yaml:"gallery"`
	Albums      AlbumsConfig       `yaml:"albums"`
	Auth        AuthConfig         `yaml:"auth"`
	Derivatives []DerivativeConfig `yaml:"derivatives"`
	Sync        SyncConfig         `yaml:"sync"`
}
type ServerConfig struct {
	Addr     string `yaml:"addr"`
	Timeouts struct {
		Read  time.Duration `yaml:"read"`
		Write time.Duration `yaml:"write"`
		Idle  time.Duration `yaml:"idle"`
	} `yaml:"timeouts"`
}

// Originals: read-only filesystem
// Derivatives: writable cache
type MediaConfig struct {
	Originals   string `yaml:"originals"`
	Derivatives string `yaml:"derivatives"`
}

type GalleryConfig struct {
	Templates struct {
		Custom string `yaml:"custom"`
	} `yaml:"templates"`
}

type AlbumsConfig struct {
	Rebuild struct {
		BatchSize   int `yaml:"batch_size"`
		Parallelism int `yaml:"parallelism"`
	} `yaml:"rebuild"`
}

type AuthConfig struct {
	Mode    AuthMode    `yaml:"mode"`
	Forward AuthForward `yaml:"forward"`
	OIDC    AuthOIDC    `yaml:"oidc"`
	JWT     JWTConfig   `yaml:"jwt"`
}

type AuthForward struct {
	UserHeader   string   `yaml:"user_header"`
	GroupsHeader string   `yaml:"groups_header"`
	TrustedCIDRs []string `yaml:"trusted_proxy_cidr"`
	AdminRole    string   `yaml:"admin_role"`
}

type AuthOIDC struct {
	Issuer    string `yaml:"issuer"`
	ClientID  string `yaml:"client_id"`
	AdminRole string `yaml:"admin_role"`
}
type JWTConfig struct {
	Secret string `yaml:"secret"`
}

type DerivativeConfig struct {
	Name      string           `yaml:"name"`
	Postfix   string           `yaml:"postfix"`
	MaxWidth  int              `yaml:"max_width"`
	MaxHeight int              `yaml:"max_height"`
	Mode      DerivateSizeMode `yaml:"mode"` // crop | fit
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type SyncConfig struct {
	Paths                []PathFilterConfig  `yaml:"paths"`
	Extensions           []string            `yaml:"extensions"` // pl: ["jpg","jpeg","png","tif","tiff","heic"]
	Metadata             MetadataConfig      `yaml:"metadata"`
	Exiftool             ExiftoolConfig      `yaml:"exiftool"`
	NormalizedExtensions map[string]struct{} `yaml:"-"`
	MergedMetadata       MetadataConfig      `yaml:"-"`
}

type PathFilterConfig struct {
	Path    string                 `yaml:"path"`    // real FS path (prefix)
	Filters ruleengine.FilterGroup `yaml:"filters"` // a már meglévő DSL
}

type MetadataConfig struct {
	Fields map[string]MetadataFieldConfig `yaml:"fields"`
}

type MetadataFieldConfig struct {
	Sources []MetadataSourceConfig `yaml:"sources"`
	Type    data.MetadataType      `yaml:"type,omitempty"`
	Unit    string                 `yaml:"unit,omnitempty"`
}

type MetadataSourceConfig struct {
	Ref string `yaml:"ref"`
}

type ExiftoolConfig struct {
	Path         string        `yaml:"path"`    // pl: "/usr/bin/exiftool"
	Timeout      time.Duration `yaml:"timeout"` // opcionális
	ResolvedPath string        `yaml:"-"`
}

func (m *MetadataSourceConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		m.Ref = value.Value
		return nil
	}
	return fmt.Errorf("invalid metadata source")
}

func Load(path string) (*Config, error) {
	log.Logger.Debug().Msg("Configuration loading start")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.Env = LoadEnvironment()
	if err := cfg.TransformBeforeValidation(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.TransformAfterValidation(); err != nil {
		return nil, err
	}

	log.Logger.Info().Msg("Configuration loaded")
	return &cfg, nil
}

func logConfigOK(path string, value any) {
	log.Logger.Info().
		Str("config", path).
		Interface("value", value).
		Msg("config set")
}

func logConfigError(path string, value any, err error) {
	log.Logger.Error().
		Str("config", path).
		Interface("value", value).
		Err(err).
		Msg("invalid config value")
}

func GetLogConfigPath() string {
	logConfig := os.Getenv(LogConfigEnv)
	if logConfig == "" {
		log.Fatal().
			Msg(LogConfigEnv + " must be set (logging config required)")
	}
	return logConfig
}
