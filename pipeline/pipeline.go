package pipeline

import (
	"context"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog/log"
)

func RunForcedImageSync(ctx context.Context, cfg config.Config, imageIDs []uint64) error {
	pipelineCtx := createPipelineContex(cfg)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pipelineCtx.Ctx = ctx
	pipelineCtx.Cancel = cancel
	log.Logger.Info().Msg("Image Force Sync Start")
	log.Logger.Debug().Object("context", logging.WithLevel(log.Logger.GetLevel(), &pipelineCtx)).Msg("with context")

	ch := make(chan WorkItem, 128)

	go func() {
		defer close(ch)
		pc := pipelineCtx
		pc.Out = ch
		_ = fSWorker(&pc)
	}()

	err := runPipeline(
		pipelineCtx,
		ch,
		/*
			stepDBLookup,

			stepHash,
		*/
		stepMetadataRead,
		// stepUpsert,
	)
	return err
}

func RunGlobalSync(ctx context.Context, cfg config.Config, cleanUp bool) error {
	mode := dbo.SyncModeFull
	if !cleanUp {
		mode = dbo.SyncModeIncremental
	}
	pipelineCtx := createPipelineContex(cfg)

	syncId, err := dao.CreateSyncRun(pipelineCtx.Database, ctx, mode)
	if err != nil {
		return err
	}
	pipelineCtx.SyncId = syncId

	cancelCtx, cancel := context.WithCancel(ctx)

	defer cancel()
	pipelineCtx.Ctx = cancelCtx
	pipelineCtx.Cancel = cancel
	log.Logger.Info().Msg("Global Sync Start")
	log.Logger.Debug().Object("context", logging.WithLevel(log.Logger.GetLevel(), &pipelineCtx)).Msg("with context")

	ch := make(chan WorkItem, 128)

	go func() {
		defer close(ch)

		pc := pipelineCtx
		pc.Out = ch
		err := fSWorker(&pc)
		if err != nil {
			cancel()
		}
	}()

	err = runPipeline(
		pipelineCtx,
		ch,
		stepDBLoopupByPath,
		stepDirtyCheck,
		stepMetadataRead,
		stepFilter,
		stepDBUpdate,
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Global Sync Failed")
		err = dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
		if err != nil {
			log.Logger.Error().Err(err).Msg("Sync close in database")
		}
		return err
	}
	if cleanUp {
		err = dao.DeleteImagesNotSeenAll(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 1000)
		if err != nil {
			log.Logger.Error().Err(err).Msg("Global Sync Failed")
			err = dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
			if err != nil {
				log.Logger.Error().Err(err).Msg("Sync close in database")
			}
			return err
		}
	}

	err = dao.CloseSyncRunSuccess(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 0, 0)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Global Sync Failed")
	}

	log.Logger.Info().Msg("Global Sync End")
	return err
}

func createPipelineContex(cfg config.Config) PipelineContext {
	return PipelineContext{
		RootPath:   cfg.Media.Originals,
		PathConfig: &cfg.Sync.Paths,
		AllowedExt: cfg.Sync.NormalizedExtensions,
		Filters:    cfg.Sync.Paths,

		Database: db.GetDatabase(),
		Metadata: &cfg.Sync.MergedMetadata,
	}
}

func runPipeline(ctx PipelineContext, input chan WorkItem, workers ...step) error {
	var err error
	for _, w := range workers {
		input, err = w(ctx, input)
		if err != nil {
			return err
		}
	}
	// Sink: drain, hogy a goroutine-ok lefussanak
	for range input {
	}
	return nil
}
