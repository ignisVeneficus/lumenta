package config

import "sync"

var (
	global *Config
	once   sync.Once
)

func SetGlobal(cfg *Config) {
	once.Do(func() {
		global = cfg
	})
}

func Global() *Config {
	if global == nil {
		panic("config not initialized")
	}
	return global
}
