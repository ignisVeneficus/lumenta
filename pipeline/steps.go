package pipeline

import (
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

			ctx.Cancel()
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

			ctx.Cancel()
			return
		}
	}()

	return out, nil
}

func stepMetadataRead(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := metadataReaderWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("metadata extractor worker failed")

			ctx.Cancel()
			return
		}
	}()

	return out, nil
}

func stepFilter(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

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

			ctx.Cancel()
			return
		}
	}()

	return out, nil
}
func stepDBUpdate(ctx PipelineContext, in chan WorkItem) (chan WorkItem, error) {

	out := make(chan WorkItem, 128)

	go func() {
		defer close(out)

		pc := ctx
		pc.In = in
		pc.Out = out

		if err := dBUpdateWorker(&pc); err != nil {

			log.Logger.Error().
				Object("pipeline", logging.WithLevel(zerolog.DebugLevel, &pc)).
				Err(err).
				Msg("DB Update worker failed")

			ctx.Cancel()
			return
		}
	}()

	return out, nil
}
