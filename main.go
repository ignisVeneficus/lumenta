package main

import (
	"context"
	"os"

	"github.com/ignisVeneficus/lumenta/cli"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/derivative"
	"github.com/ignisVeneficus/lumenta/exif"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	logging.LoadLogging(config.GetLogConfigPath())

	cfgPath := os.Getenv("LUMENTA_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to load configuration")
		panic(err)
	}
	config.SetGlobal(cfg)

	derivative.Init(context.Background(), 10)
	defer derivative.Shutdown()
	defer exif.ShutdownExiftool()

	err = cli.Run(*cfg)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to run Lumenta")
		panic(err)
	}

}
