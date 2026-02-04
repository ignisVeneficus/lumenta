package server

import (
	"time"
)

type ServerConfig struct {
	Addr     string `yaml:"addr"`
	Timeouts struct {
		Read  time.Duration `yaml:"read"`
		Write time.Duration `yaml:"write"`
		Idle  time.Duration `yaml:"idle"`
	} `yaml:"timeouts"`
}
