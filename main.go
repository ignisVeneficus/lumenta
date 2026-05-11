package main

import (
	"context"
	"os"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/cli"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/derivative"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	logging.LoadLogging(config.GetLogConfigPath())

	cfgPath := os.Getenv("LUMENTA_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	logScope, ctx := logging.Enter(ctx, "main", nil, nil)
	cfg, err := config.Load(cfgPath, ctx)
	if err != nil {
		logging.Fatal(logScope, "load configuration", nil, err, "")
		panic(err)
	}
	config.SetGlobal(cfg)

	i18n, err := i18n.Init()
	if err != nil {
		logging.Fatal(logScope, "load i18n", nil, err, "")
		panic(err)
	}

	derivative.Init(ctx, 10)
	defer derivative.Shutdown(ctx)

	database := db.GetDatabaseMulti()
	defer database.Close()

	if err := dao.CreateDatabase(database, ctx); err != nil {
		logging.Fatal(logScope, "connect to database", nil, err, "")
		log.Logger.Info().Msg("Stopping")
		panic(err)
	}

	err = cli.Run(*cfg, i18n, ctx)
	if err != nil {
		logging.Fatal(logScope, "run cli", nil, err, "")
		log.Logger.Info().Msg("Stopping")
		panic(err)
	}

}
