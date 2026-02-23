package pipeline

import (
	"context"
	"errors"

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
	logg := logging.Enter(ctx, "image.sync.global", map[string]any{"root": cfg.Filesystem.Originals, "cleanup": cleanUp})
	pipelineCtx := createPipelineContex(cfg)
	metaHash := cfg.Sync.MetadataHash
	dbMetaHash, err := dao.GetSyncRunLastHash(pipelineCtx.Database, ctx)
	if err != nil {
		if errors.Is(err, dao.ErrDataNotFound) {
			dbMetaHash = ""
		} else {
			logging.ExitErr(logg, err)
			return err
		}
	}
	mode := dbo.SyncModeFull
	switch {
	case !cleanUp:
		mode = dbo.SyncModeIncremental
	case metaHash != dbMetaHash:
		pipelineCtx.Force = true
	}

	syncId, err := dao.CreateSyncRun(pipelineCtx.Database, ctx, mode, metaHash)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	pipelineCtx.SyncId = syncId

	cancelCtx, cancel := context.WithCancel(ctx)

	defer cancel()
	pipelineCtx.Ctx = cancelCtx
	pipelineCtx.Cancel = cancel
	logging.Info("image.sync.global", "start", "", "", nil)
	logging.Inside(logg, map[string]any{"context": pipelineCtx}, "contex.created")

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
		stepACL,
		stepDBUpdate,
	)
	if err != nil {
		logging.ExitErr(logg, err)
		err = dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
		if err != nil {
			logging.ExitErr(logg, err)
		}
		return err
	}
	if cleanUp {
		err = dao.DeleteImagesNotSeenAll(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 1000)
		if err != nil {
			logging.ExitErr(logg, err)
			err = dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
			if err != nil {
				logging.ExitErr(logg, err)
			}
			return err
		}
	}

	err = dao.CloseSyncRunSuccess(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 0, 0)
	if err != nil {
		logging.ExitErr(logg, err)
	}

	logging.Exit(logg, "ok", nil)
	return err
}

func createPipelineContex(cfg config.Config) PipelineContext {
	return PipelineContext{
		RootPath:    cfg.Filesystem.Originals,
		AllowedExt:  cfg.Sync.NormalizedExtensions,
		Filters:     cfg.Sync.Paths,
		ACLRules:    cfg.Sync.ACLRules,
		ACLOverride: cfg.Sync.ACLOverride,

		Database: db.GetDatabase(),
		Metadata: &cfg.Sync.MergedMetadata,
		Panorama: cfg.Sync.Panorama,
		Force:    false,
	}
}

func runPipeline(ctx PipelineContext, input chan WorkItem, workers ...step) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.pipeline", nil)
	var err error
	for _, w := range workers {
		input, err = w(ctx, input)
		if err != nil {
			logging.ExitErr(logg, err)
			return err
		}
	}
	// Sink: drain, hogy a goroutine-ok lefussanak
	for range input {
	}
	return nil
}
