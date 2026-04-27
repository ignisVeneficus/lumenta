package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
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
	var (
		seen    uint64 = 0
		notSeen uint64 = 0
		rt      *SyncRuntime
	)
	defer func() {
		if pipelineCtx.SyncId == 0 {
			return // not started
		}
		if rt != nil {
			rt.Stop(pipelineCtx.SyncId)
		}
		if err != nil {
			logging.ExitErr(logg, err)
			cerr := dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
			if cerr != nil {
				logging.ExitErr(logg, cerr)
			}
		} else {
			cerr := dao.CloseSyncRunSuccess(pipelineCtx.Database, ctx, pipelineCtx.SyncId, seen, notSeen)
			if cerr != nil {
				logging.ExitErr(logg, cerr)
				err = cerr
			}
		}

	}()

	// FIXME: hash always empty
	syncId, err := dao.CreateSyncRun(pipelineCtx.Database, ctx, mode, metaHash)
	if err != nil {
		return err
	}
	pipelineCtx.SyncId = syncId

	rt = Global()

	if !rt.Start(syncId) {
		return fmt.Errorf("sync already running")
	}

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
		return err
	}
	seen, err = dao.CountImageByLastSeen(pipelineCtx.Database, ctx, pipelineCtx.SyncId)
	if err != nil {
		return err
	}
	if cleanUp {
		// delete only is cleanup set
		notSeen, err = dao.CountImageByLastNotSeen(pipelineCtx.Database, ctx, pipelineCtx.SyncId)
		if err != nil {
			return err
		}

		err = dao.DeleteImageNotSeenAll(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 1000)
		if err != nil {
			return err
		}
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
	// Sink: drain
	for range input {
	}
	return nil
}

type writeReason string

const (
	reasonError   writeReason = "error"
	reasonSkipped writeReason = "skipped"
	reasonOk      writeReason = "ok"
)

func SaveResultSkip(px *PipelineContext, job WorkItem) error {
	return saveResult(px, job, reasonSkipped)
}
func SaveResultSucess(px *PipelineContext, job WorkItem) error {
	return saveResult(px, job, reasonOk)
}
func SaveResultError(px *PipelineContext, job WorkItem) error {
	return saveResult(px, job, reasonError)
}

func saveResult(px *PipelineContext, job WorkItem, reason writeReason) error {
	dbItem := dbo.SyncFile{
		SyncID:   px.SyncId,
		Root:     job.RootName,
		Path:     job.Path,
		Filename: job.Filename,
		Ext:      job.Ext,
	}
	if job.RuleResults != nil {
		ruleResultsJSON, err := json.Marshal(job.RuleResults)
		if err != nil {
			return err
		}
		dbItem.RuleResultsJSON = ruleResultsJSON
	}
	if job.DirtyReason != "" {
		dbItem.DirtyReason = (*string)(&job.DirtyReason)
	}
	switch reason {
	case reasonSkipped:
		if job.DBImage.ID != nil {
			dbItem.Status = dbo.SyncFileStatusDeleted
		} else {
			dbItem.Status = dbo.SyncFileStatusFilteredOut
		}
	case reasonOk:
		switch job.DirtyReason {
		case data.DirtyHashChg, data.DirtyMetadataHashChg,
			data.DirtySizeChg, data.DirtyTimeChg:
			dbItem.Status = dbo.SyncFileStatusUpdated
		case data.DirtyNewfile:
			dbItem.Status = dbo.SyncFileStatusCreated
		case data.DirtyForced:
			if job.DBImage.ID != nil {
				dbItem.Status = dbo.SyncFileStatusUpdated
			} else {
				dbItem.Status = dbo.SyncFileStatusCreated
			}
		default:
			dbItem.Status = dbo.SyncFileStatusNotChanged
		}
	case reasonError:
		dbItem.Status = dbo.SyncFileStatusError
	}
	err := dao.CreateSyncFile(px.Database, px.Ctx, dbItem)
	return err

}
