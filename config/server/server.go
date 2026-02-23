package server

import (
	"time"
)

type ServerConfig struct {
	Addr     string `yaml:"addr"`
	Timeouts struct {
		Read   time.Duration `yaml:"read"`
		Write  time.Duration `yaml:"write"`
		Header time.Duration `yaml:"readHeader"`
		Idle   time.Duration `yaml:"idle"`
	} `yaml:"timeouts"`
}
