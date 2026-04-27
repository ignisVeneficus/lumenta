package main

import (
	"context"
	"os"

	"github.com/ignisVeneficus/lumenta/cli"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/derivative"
	"github.com/ignisVeneficus/lumenta/exif"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
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

	i18n, err := i18n.Init()
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to load i18n")
		panic(err)
	}

	derivative.Init(context.Background(), 10)
	defer derivative.Shutdown()
	defer exif.ShutdownExiftool()

	database := db.GetDatabaseMulti()
	defer database.Close()
	ctx := context.Background()
	if err := dao.CreateDatabase(database, ctx); err != nil {
		log.Logger.Fatal().Err(err).Msg("database Error")
		log.Logger.Info().Msg("Stopping")
		panic(err)
	}

	err = cli.Run(*cfg, i18n)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to run Lumenta")
		log.Logger.Info().Msg("Stopping")
		panic(err)
	}

}
