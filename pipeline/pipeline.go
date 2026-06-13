package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog/log"
)

func RunForcedImageSync(ctx context.Context, cfg config.Config, imageIDs []uint64) error {
	pipelineCtx := createPipelineContex(cfg, ctx)
	ctx, cancel := context.WithCancelCause(ctx)
	var err error
	defer cancel(err)
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

	out, err := runPipeline(
		pipelineCtx,
		ch,
		/*
			stepDBLookup,

			stepHash,
		*/
		stepMetadataReader,
		// stepUpsert,
	)

	for range out {
	}

	return err
}

func RunGlobalSync(c context.Context, cfg config.Config, cleanUp bool, force bool) error {
	logScope, ctx := logging.Enter(c, "sync/global", nil, map[string]any{"root": cfg.Filesystem.Originals, "cleanup": cleanUp})
	pipelineCtx := createPipelineContex(cfg, ctx)
	metaHash := cfg.Sync.MetadataHash
	dbMetaHash, err := dao.GetSyncRunLastHash(pipelineCtx.Database, ctx)
	if err != nil {
		if errors.Is(err, dao.ErrDataNotFound) {
			dbMetaHash = ""
		} else {
			logging.ExitErr(logScope, err)
			return err
		}
	}
	mode := dbo.SyncModeFull
	switch {
	case !cleanUp:
		mode = dbo.SyncModeIncremental
	case metaHash != dbMetaHash:
		pipelineCtx.Force = true
	case force:
		pipelineCtx.Force = true
	}
	var (
		seen    uint64 = 0
		notSeen uint64 = 0
		rt      *SyncRuntime
	)
	defer func() {
		if pipelineCtx.SyncId == 0 {
			logging.Exit(logScope, "not started", nil)
			return // not started
		}
		if rt != nil {
			rt.Stop(uint64(pipelineCtx.SyncId))
		}
		if err != nil {
			logging.ExitErr(logScope, err)
			cerr := dao.CloseSyncRunError(pipelineCtx.Database, ctx, pipelineCtx.SyncId, err.Error())
			if cerr != nil {
				logging.ExitErr(logScope, cerr)
			}
		} else {
			cerr := dao.CloseSyncRunSuccess(pipelineCtx.Database, ctx, pipelineCtx.SyncId, seen, notSeen)
			if cerr != nil {
				logging.ExitErr(logScope, cerr)
				err = cerr
			}
		}

	}()

	syncId, err := dao.CreateSyncRun(pipelineCtx.Database, ctx, mode, metaHash)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	pipelineCtx.SyncId = syncId

	rt = Global()

	if !rt.Start(uint64(syncId)) {
		err = fmt.Errorf("sync already running")
		logging.ExitErr(logScope, err)
		return err
	}

	cancelCtx, cancel := context.WithCancelCause(ctx)

	defer func() {
		cancel(err)
	}()
	pipelineCtx.Ctx = cancelCtx
	pipelineCtx.Cancel = cancel
	logging.Info(logScope, "start", nil)
	logging.Debug(logScope, "ctx created", map[string]any{"context": pipelineCtx})

	ch := make(chan WorkItem, 128)
	filterOut := make(chan WorkItem, 128)

	pipelineCtx.FilterOut = filterOut

	wg := &sync.WaitGroup{}
	pipelineCtx.WG = wg

	go func() {
		defer close(ch)

		pc := pipelineCtx
		pc.Out = ch
		err := fSWorker(&pc)
		if err != nil {
			cancel(err)
		}
	}()
	// run pipeline
	out, err := runPipeline(
		pipelineCtx,
		ch,
		stepHash,
		stepDBLoopupByPath,
		stepDirtyCheck,
		stepMetadataReader,
		stepFilter,
		stepACL,
		stepDBImageWriter,
		stepAlbumInsertion,
		stepResultSaver,
	)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	go func() {
		wg.Wait()
		close(filterOut)
	}()
	// filtered out pipeline
	filterCtx := pipelineCtx
	filterCtx.FilterOut = nil
	var filteredOut <-chan WorkItem
	filteredOut, err = runPipeline(
		filterCtx,
		filterOut,
		stepDBFilterWriter,
	)
	// Sink: drain
	drain(out, filteredOut)

	err = context.Cause(cancelCtx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	seen, err = dao.CountImageByLastSeen(pipelineCtx.Database, ctx, pipelineCtx.SyncId)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	err = dao.ReorderAllImages(pipelineCtx.Database, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	if cleanUp {
		// delete only is cleanup set
		notSeen, err = dao.CountImageByLastNotSeen(pipelineCtx.Database, ctx, pipelineCtx.SyncId)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}

		err = dao.DeleteImageNotSeenAll(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 1000)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		err = dao.DeleteFilteredNotSeenAll(pipelineCtx.Database, ctx, pipelineCtx.SyncId, 1000)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
	}

	logging.Exit(logScope, "ok", nil)
	return err
}

func collectAlbums(database *sql.DB, c context.Context) (*AlbumContext, error) {
	logScope, ctx := logging.Enter(c, "sync/pipeline/albums", nil, nil)
	albums, err := dao.QueryAlbum(database, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	paths := make(map[uint64]string)
	rules := make([]*AlbumRule, len(albums))
	albumStuct := make(ruleengine.AlbumsStruct, len(albums))
	albumIDs := make(map[uint64]*AlbumRule, len(albums))
	for i, a := range albums {
		id := (*uint64)(a.ID)
		parentID := (*uint64)(a.ParentID)
		pathId := make([]uint64, len(a.AncestorIDs))
		for j, pid := range a.AncestorIDs {
			pathId[j] = uint64(pid)
		}
		albumrule := &AlbumRule{
			ID:       *id,
			PathIDs:  pathId,
			Depth:    len(a.AncestorIDs),
			Name:     a.Name,
			Rank:     a.Rank,
			ParentID: parentID,
		}
		rules[i] = albumrule
		albumIDs[*id] = albumrule

		var rawRule ruleengine.RuleGroup
		err := json.Unmarshal(a.RuleJSON, &rawRule)
		if err != nil {
			logging.ErrorContinue(logScope, err, map[string]any{
				"album_id":   a.ID,
				"album_name": a.Name,
			})
			continue
		}
		rule, err := ruleengine.CompileGroupFilter(rawRule, fmt.Sprintf("%s (%d)", a.Name, *a.ID))
		if err != nil {
			if errors.Is(err, ruleengine.ErrEmptyFilter) {
				logging.Debug(logScope, "empty rule", map[string]any{
					"album_id":   a.ID,
					"album_name": a.Name,
				})
			} else {
				logging.ErrorContinue(logScope, err, map[string]any{
					"album_id":   a.ID,
					"album_name": a.Name,
				})
			}
			continue
		}
		albumrule.Rule = rule
		logging.Debug(logScope, "compiled", map[string]any{
			"album_id":   a.ID,
			"album_name": a.Name,
		})
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Depth < rules[j].Depth
	})

	for _, a := range rules {
		if a.ParentID == nil {
			a.RankOrder = []uint64{a.Rank}
			set := map[uint64]struct{}{
				a.ID: {},
			}
			albumStuct[a.ID] = set
			paths[a.ID] = a.Name
			continue
		}
		parent := albumIDs[*a.ParentID]
		parentPath := paths[*a.ParentID]

		path := make([]uint64, parent.Depth+1)
		copy(path, parent.RankOrder)
		path[parent.Depth] = a.Rank
		set := make(map[uint64]struct{}, a.Depth)
		for _, id := range a.PathIDs {
			set[id] = struct{}{}
		}
		albumStuct[a.ID] = set
		a.RankOrder = path
		paths[a.ID] = parentPath + "/" + a.Name
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Depth != rules[j].Depth {
			return rules[i].Depth > rules[j].Depth
		}
		for k, r := range rules[i].RankOrder {
			if r < rules[j].RankOrder[k] {
				return true
			}
			if r > rules[j].RankOrder[k] {
				return false
			}
		}
		return rules[i].ID < rules[j].ID
	})
	logging.Exit(logScope, "ok", nil)
	return &AlbumContext{
		NameMap:      paths,
		AlbumStructs: albumStuct,
		Rules:        rules,
	}, nil
}

func createPipelineContex(cfg config.Config, ctx context.Context) PipelineContext {
	database := db.GetDatabase()
	albumCtx, err := collectAlbums(database, ctx)
	if err != nil {
		albumCtx = nil
	}
	pipelineContext := PipelineContext{
		RootPath:       cfg.Filesystem.Originals,
		AllowedExt:     cfg.Sync.NormalizedExtensions,
		Filters:        cfg.Sync.Paths,
		ACLRules:       cfg.Sync.ACLRules,
		ACLOverride:    cfg.Sync.ACLOverride,
		ExifToolConfig: cfg.Sync.Exiftool,
		Workers:        cfg.Sync.Pipeline,

		Database: database,
		Metadata: &cfg.Sync.MergedMetadata,
		Panorama: cfg.Sync.Panorama,
		Force:    false,
		AlbumCtx: albumCtx,
	}

	return pipelineContext
}

func runPipeline(c PipelineContext, input chan WorkItem, workers ...step) (chan WorkItem, error) {
	logScope, _ := logging.Enter(c.Ctx, "sync/pipeline/run", nil, nil)
	var err error
	for _, w := range workers {
		input, err = w(c, input)
		if err != nil {
			logging.ExitErr(logScope, err)
			return input, err
		}
	}
	return input, nil
}

type writeReason string

const (
	reasonError   writeReason = "error"
	reasonSkipped writeReason = "skipped"
	reasonOk      writeReason = "ok"
)

func SaveResultSkip(px *PipelineContext, job WorkItem, ct context.Context) error {
	return saveResult(px, job, ct, reasonSkipped)
}
func SaveResultSucess(px *PipelineContext, job WorkItem, ct context.Context) error {
	return saveResult(px, job, ct, reasonOk)
}
func SaveResultError(px *PipelineContext, job WorkItem, ct context.Context) error {
	return saveResult(px, job, ct, reasonError)
}

func saveResult(px *PipelineContext, job WorkItem, ct context.Context, reason writeReason) error {
	logScope, c := logging.Enter(ct, "pipeline/save_result", job.RealPath, map[string]any{
		"path":   job.RealPath,
		"reason": reason,
		"source": job.Source,
	})
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
			logging.ExitErr(logScope, err)
			return err
		}
		dbItem.RuleResultsJSON = ruleResultsJSON
	}
	if job.DirtyReason != "" {
		dbItem.DirtyReason = (*string)(&job.DirtyReason)
	}
	switch reason {
	case reasonSkipped:
		if job.Source == SourceImages {
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
	err := dao.CreateSyncFile(px.Database, c, dbItem)
	return logging.Return(logScope, err)

}
func drain(a <-chan WorkItem, b <-chan WorkItem) {
	for a != nil || b != nil {
		select {
		case w, ok := <-a:
			if !ok {
				a = nil
				continue
			}
			logging.Return(w.LogScope, w.Err)

		case w, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			logging.Return(w.LogScope, w.Err)
		}
	}
}
