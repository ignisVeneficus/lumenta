package pipeline

import (
	"sync"

	"github.com/ignisVeneficus/logging"
	syncConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
)

const (
	SizeOfPipeline = 128
)

type step func(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error)

func stepDBLoopupByPath(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/db_lookup/build", nil, nil)
	out := make(chan WorkItem, 128)

	go func() {
		logScope, _ := logging.Enter(c, "sync/pipeline/db_lookup/run", nil, nil)
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dBLoopupByPathWorker(&pc); err != nil {
			logging.ExitErr(logScope, err)
			ctx.Cancel(err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}

func stepDirtyCheck(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/dirty_check/build", nil, nil)

	out := make(chan WorkItem, 128)
	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepDirty]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/dirty_check/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := dirtyCheckWorker(&pc); err != nil {

				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}

func stepHash(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/hash/build", nil, nil)
	out := make(chan WorkItem, 128)

	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepHash]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/hash/run", i, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := hashWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}

func stepMetadataReader(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/metadat_reader/build", nil, nil)

	out := make(chan WorkItem, 128)

	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepMetadata]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/metadat_reader/run", i, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := metadataReaderWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}

func stepFilter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/import_filter/build", nil, nil)
	out := make(chan WorkItem, 128)

	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepFilter]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)
	if ctx.FilterOut != nil {
		ctx.WG.Add(workers)
	}

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/import_filter/run", nil, map[string]any{
				"index": i,
			})
			defer func() {
				if ctx.FilterOut != nil {
					ctx.WG.Done()
				}
				wg.Done()
			}()
			if err := filterWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}
func stepDBImageWriter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/image_writer/build", nil, nil)

	tagCache := CreateTagCache()
	err := LoadTagCache(&tagCache, ctx.Database, c)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	out := make(chan WorkItem, 128)
	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepImage]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/image_writer/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := dbImageWriterWorker(&pc, &tagCache); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}

func stepACL(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/acl_rules/build", nil, map[string]any{
		"acl_rules": len(ctx.ACLRules),
	})
	if len(ctx.ACLRules) == 0 {
		logging.Exit(logScope, "not need", nil)
		return in, nil
	}
	out := make(chan WorkItem, 128)
	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepACL]; ok {
		workers = int(stepConfig.Workers)
	}
	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/acl_rules/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := aclWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}
func stepAlbumInsertion(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/album_rules/build", nil, nil)
	database := db.GetDatabase()
	albumQty, err := dao.CountAlbum(database, c)
	if err != nil {
		logging.ExitErr(logScope, err)
		ctx.Cancel(err)
		return nil, err
	}
	if albumQty == 0 {
		logging.Exit(logScope, "not need", nil)
		return in, nil
	}
	out := make(chan WorkItem, 128)

	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepAlbum]; ok {
		workers = int(stepConfig.Workers)
	}
	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/album_rules/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := albumInsertionWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}
func stepResultSaver(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/result_saver/build", nil, nil)
	out := make(chan WorkItem, 128)

	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepResult]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {

		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/result_saver/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := resultSaverWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}
func stepDBFilterWriter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	logScope, c := logging.Enter(ctx.Ctx, "sync/pipeline/filtered_writer/build", nil, nil)

	out := make(chan WorkItem, 128)
	pc := ctx
	pc.In = in
	pc.Out = out

	workers := 1
	if stepConfig, ok := ctx.Workers[syncConfig.StepFiltered]; ok {
		workers = int(stepConfig.Workers)
	}

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			logScope, _ := logging.Enter(c, "sync/pipeline/filtered_writer/run", nil, map[string]any{
				"index": i,
			})
			defer wg.Done()

			if err := dbFilteredWriterWorker(&pc); err != nil {
				logging.ExitErr(logScope, err)
				ctx.Cancel(err)
				return
			}
			logging.Exit(logScope, "ok", nil)
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	logging.Exit(logScope, "end", nil)
	return out, nil
}
