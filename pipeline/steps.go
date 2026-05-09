package pipeline

import (
	"context"
	"sync"

	syncConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	SizeOfPipeline = 128
)

type step func(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error)

func stepDBLoopupByPath(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dBLoopupByPathWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("Db lookup worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}

func stepDirtyCheck(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dirtyCheckWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("dirty check worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}

func stepHash(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
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
			defer wg.Done()

			if err := hashWorker(&pc); err != nil {

				log.Logger.Error().
					Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
					Err(err).
					Msg("hash calculation worker failed")

				ctx.Cancel(err)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}

func stepMetadataReader(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

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
			defer wg.Done()

			if err := metadataReaderWorker(&pc); err != nil {

				log.Logger.Error().
					Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
					Err(err).
					Msg("metadata extractor worker failed")

				ctx.Cancel(err)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}

func stepFilter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)
	if ctx.FilterOut != nil {
		ctx.WG.Add(1)
	}
	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := filterWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("filter worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}
func stepDBImageWriter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dbImageWriterWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("DB Update worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}

func stepACL(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	if len(ctx.ACLRules) == 0 {
		return in, nil
	}

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := aclWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("acl worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}
func stepAlbumInsertion(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	database := db.GetDatabase()
	albumQty, err := dao.CountAlbum(database, context.Background())
	if err != nil {
		return nil, err
	}
	if albumQty == 0 {
		return in, nil
	}
	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := albumInsertionWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("album worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}
func stepResultSaver(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {
	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := resultSaverWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("result save worker failed")

			ctx.Cancel(err)
			return
		}
	}()

	return out, nil
}
func stepDBFilterWriter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dbFilteredWriterWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("filtered worker failed")

			ctx.Cancel(err)
			return
		}
	}()
	return out, nil
}
