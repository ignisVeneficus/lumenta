package config

import (
	"os"

	"github.com/ignisVeneficus/lumenta/config/auth"
	"github.com/ignisVeneficus/lumenta/config/database"
	"github.com/ignisVeneficus/lumenta/config/derivative"
	"github.com/ignisVeneficus/lumenta/config/filesystem"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/config/server"
	"github.com/ignisVeneficus/lumenta/config/site"
	"github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	ENV_PREFIX = "LUMENTA"
)

var (
	LogConfigEnv = ENV_PREFIX + "_LOG_CONFIG"
)

type Config struct {
	Env          Environment                     `yaml:"-"`
	Server       server.ServerConfig             `yaml:"server"`
	Database     database.DatabaseConfig         `yaml:"database"`
	Filesystem   filesystem.FilesystemConfig     `yaml:"filesystem"`
	Auth         auth.AuthConfig                 `yaml:"auth"`
	Derivatives  derivative.DerivativesConfig    `yaml:"derivatives"`
	Sync         sync.SyncConfig                 `yaml:"sync"`
	Site         site.SiteConfig                 `yaml:"site"`
	Presentation presentation.PresentationConfig `yaml:"presentation"`
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

func GetLogConfigPath() string {
	logConfig := os.Getenv(LogConfigEnv)
	if logConfig == "" {
		log.Fatal().
			Msg(LogConfigEnv + " must be set (logging config required)")
	}
	return logConfig
}
