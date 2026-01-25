package pipeline

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/mapper"
	"github.com/ignisVeneficus/lumenta/metadata"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func walkDirHandler(ctx *PipelineContext, realPath string, d fs.DirEntry, err error) error {

	if err != nil {
		return err
	}

	select {
	case <-ctx.Ctx.Done():
		return ctx.Ctx.Err()
	default:
	}

	if d.IsDir() {
		return nil
	}
	info, err := d.Info()
	if err != nil {
		return nil
	}
	realPath = filepath.ToSlash(realPath)
	fullPath := strings.TrimPrefix(strings.TrimPrefix(realPath, ctx.RootPath), "/")

	path, filename, ext := utils.SplitPath(fullPath)
	normalisedExt := utils.NormalizeExt(ext)
	if len(ctx.AllowedExt) > 0 {
		if _, ok := ctx.AllowedExt[normalisedExt]; !ok {
			return nil
		}
	}

	fileHash, err := utils.ComputeFileHash(realPath)
	if err != nil {
		return nil
	}
	metaHash, err := utils.ComputeFileHash(realPath + ".xmp")
	if err != nil {
		if os.IsNotExist(err) {
			metaHash = ""
		} else {
			return err
		}
	}

	if ctx.Out == nil {
		return nil
	}

	ctx.Out <- WorkItem{
		RootPath:         ctx.RootPath,
		Path:             path,
		RealPath:         realPath,
		Ext:              normalisedExt,
		Filename:         filename,
		Info:             info,
		FileHash:         fileHash,
		FileMetadataHash: metaHash,
	}
	return nil
}

func fSWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.fsWalker", map[string]any{"root": ctx.RootPath})

	err := filepath.WalkDir(ctx.RootPath, func(path string, d fs.DirEntry, err error) error {
		return walkDirHandler(ctx, path, d, err)
	})
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dBLoopupByPathWorker(ctx *PipelineContext) error {
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dBLoopupByPathWorker start")
	if ctx.Database == nil {
		return fmt.Errorf("dBLoopupByPathWorker: DB is nil")
	}
	if ctx.In == nil || ctx.Out == nil {
		return fmt.Errorf("dBLoopupByPathWorker: In/Out channel is nil")
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("dBLoopupByPathWorker get a job")

		image, err := dao.GetImageByPath(ctx.Database, ctx.Ctx, job.Path, job.Filename)
		switch {
		case err == nil:
			job.DBImage = &image
			log.Logger.Info().Str("path", job.RealPath).Msg("Old image found")
		case errors.Is(err, dao.ErrDataNotFound):
			job.IsDirty = true
			job.DirtyReason = dirtyNewfile
			job.DBImage = &dbo.Image{
				FocusMode: dbo.ImageFocusModeAuto,
				ACLScope:  dbo.ACLScopePublic,
			}

			log.Logger.Info().Str("path", job.RealPath).Msg("New Image found")

		default:
			log.Logger.Error().Str("path", ctx.RootPath).Msg("dBLoopupByPathWorker died")
			return err
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("dBLoopupByPathWorker finished a job")
		ctx.Out <- job
	}
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dBLoopupByPathWorker end")
	return nil
}

func dirtyCheckWorker(ctx *PipelineContext) error {
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dirtyCheckWorker start")
	if ctx.In == nil || ctx.Out == nil {
		return fmt.Errorf("dirtyCheckWorker: In/Out channel is nil")
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			log.Logger.Error().Err(ctx.Ctx.Err()).Str("path", job.RealPath).Msg("dirtyCheckWorker died")
			return ctx.Ctx.Err()
		default:
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("dirtyCheckWorker get a job")

		if !job.IsDirty {
			for {
				if job.FileHash != job.DBImage.FileHash {
					job.IsDirty = true
					job.DirtyReason = dirtyHashChg
					break
				}
				if job.FileMetadataHash != job.DBImage.MetaHash {
					job.IsDirty = true
					job.DirtyReason = dirtyMetadataHashChg
					break
				}
				if job.DBImage.FileSize != uint64(job.Info.Size()) {
					job.IsDirty = true
					job.DirtyReason = dirtySizeChg
					break
				}
				if !job.Info.ModTime().Equal(job.DBImage.MTime) {
					job.IsDirty = true
					job.DirtyReason = dirtyTimeChg
					break
				}
				break
			}
		}
		log.Logger.Debug().Str("path", job.RealPath).Bool("is dirty", job.IsDirty).Str("dirty reaseon", string(job.DirtyReason)).Msg("dirtyCheckWorker finished a job")
		ctx.Out <- job
	}
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dirtyCheckWorker end")
	return nil
}
func metadataReaderWorker(ctx *PipelineContext) error {
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("metadataReaderWorker start")
	if ctx.In == nil || ctx.Out == nil {
		return fmt.Errorf("metadataReaderWorker: In/Out channel is nil")
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		log.Logger.Debug().Str("path", job.RealPath).Bool("is dirty", job.IsDirty).Msg("metadataReaderWorker get a job")
		if job.IsDirty {
			metadata, err := metadata.ExtractMetadata(ctx.Ctx, job.RealPath)
			if err != nil {
				log.Logger.Error().Str("path", ctx.RootPath).Msg("metadataReaderWorker cant extract data")
				continue
			}
			job.Metadata = metadata
			log.Logger.Debug().Str("path", job.RealPath).Object("metadata", logging.WithLevel(zerolog.DebugLevel, &job.Metadata)).Msg("metadataReaderWorker finished a job")
		}
		ctx.Out <- job
	}
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("metadataReaderWorker end")
	return nil
}
func filterWorker(ctx *PipelineContext) error {
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("filterWorker start")
	if ctx.In == nil || ctx.Out == nil {
		return fmt.Errorf("filterWorker: In/Out channel is nil")
	}
	filters := map[string]ruleengine.CompiledFilter{}
	for _, f := range ctx.Filters {
		key := f.Path
		filterGrp := f.Filters
		cf, err := ruleengine.CompileGroupFilter(filterGrp)
		if err != nil {
			log.Logger.Error().Err(err).Str("filter", key).Str("path", ctx.RootPath).Msg("filter cant compile")
			continue
		}
		filters[key] = cf
	}

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("filterWorker get a job")
		rating := 0
		if job.Metadata.GetRating() != nil {
			rating = int(*job.Metadata.GetRating())
		}
		facts := ruleengine.ImageFacts{
			Path:     job.Path,
			Filename: job.Filename,
			Ext:      job.Ext,
			TakenAt:  job.Metadata.GetTakenAt(),
			Rating:   &rating,
			Tags:     job.Metadata.GetTags(),
		}
		match := true
		notKey := ""
		for key, filter := range filters {
			if strings.HasPrefix(job.Path, key) {
				log.Logger.Debug().Str("path", job.RealPath).Str("value", job.Path).Str("rule", key).Msg("filterWorker check a rule")
				match = filter(facts)
				if !match {
					notKey = key
					break
				}
			}
		}
		if !match {
			log.Logger.Info().Str("path", job.RealPath).Str("rule", notKey).Msg("Filtered out")
			log.Logger.Debug().Str("path", job.RealPath).Msg("filterWorker finished a job")
			continue
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("filterWorker finished a job")
		ctx.Out <- job
	}
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("filterWorker end")
	return nil
}

func dBUpdateWorker(ctx *PipelineContext) error {
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dBUpdateWorker start")
	if ctx.In == nil || ctx.Out == nil {
		return fmt.Errorf("dBUpdateWorker: In/Out channel is nil")
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("dBUpdateWorker get a job")

		if job.IsDirty {
			job.DBImage.Path = job.Path
			job.DBImage.Filename = job.Filename
			job.DBImage.FileSize = uint64(job.Info.Size())
			job.DBImage.MTime = job.Info.ModTime()
			job.DBImage.Ext = job.Ext
			job.DBImage.FileHash = job.FileHash
			job.DBImage.MetaHash = job.FileMetadataHash
			job.DBImage.LastSeenSync = &ctx.SyncId
			mapper.UpdateImageMetadata(job.DBImage, job.Metadata)
			updateID, err := dao.CreateOrUpdateImage(ctx.Database, ctx.Ctx, job.DBImage)
			if err != nil {
				log.Logger.Error().Err(err).Bool("is dirty", job.IsDirty).Str("path", job.RealPath).Msg("database insertion/update failed")
			} else {
				job.DBImage.ID = &updateID
			}
		} else {
			err := dao.UpdateImageSyncId(ctx.Database, ctx.Ctx, *job.DBImage.ID, ctx.SyncId)
			if err != nil {
				log.Logger.Error().Err(err).Bool("is dirty", job.IsDirty).Str("path", job.RealPath).Msg("database insertion/update failed")
			}
		}
		log.Logger.Debug().Str("path", job.RealPath).Msg("dBUpdateWorker finished a job")
		ctx.Out <- job
	}
	log.Logger.Debug().Str("path", ctx.RootPath).Msg("dBUpdateWorker end")
	return nil
}
