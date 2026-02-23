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
	"github.com/rs/zerolog/log"
)

func walkDirHandler(ctx *PipelineContext, rootName, realPath string, d fs.DirEntry, err error) error {

	if err != nil {
		return err
	}

	select {
	case <-ctx.Ctx.Done():
		return ctx.Ctx.Err()
	default:
	}

	root, ok := ctx.RootPath[rootName]
	if !ok {
		return nil
	}

	realPath = filepath.ToSlash(realPath)
	fullPath := strings.TrimPrefix(strings.TrimPrefix(realPath, root.Root), "/")

	if d.IsDir() {
		for _, excl := range root.Excluded {
			if strings.HasPrefix(fullPath, excl) {
				log.Logger.Info().Str("path", realPath).Str("name", rootName).Msg("Skipped for sync")
				return filepath.SkipDir
			}
		}
		return nil
	}
	info, err := d.Info()
	if err != nil {
		return nil
	}
	path, filename, ext := utils.SplitPath(fullPath)
	normalisedExt := utils.NormalizeExt(ext)
	if len(ctx.AllowedExt) > 0 {
		if _, ok := ctx.AllowedExt[normalisedExt]; !ok {
			return nil
		}
	}
	metaFile := realPath + ".xmp"
	fileHash, err := utils.ComputeFileHash(realPath)
	if err != nil {
		return nil
	}
	metaHash, err := utils.ComputeFileHash(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			metaHash = ""
			metaFile = ""
		} else {
			return err
		}
	}

	if ctx.Out == nil {
		return nil
	}

	ctx.Out <- WorkItem{
		RootPath:         root.Root,
		RootName:         rootName,
		Path:             path,
		RealPath:         realPath,
		MetadataFile:     metaFile,
		Ext:              normalisedExt,
		Filename:         filename,
		Info:             info,
		FileHash:         fileHash,
		FileMetadataHash: metaHash,
	}
	return nil
}

func fSWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.fsWalker", nil)

	for rootName, rootConfig := range ctx.RootPath {
		err := filepath.WalkDir(rootConfig.Root, func(path string, d fs.DirEntry, err error) error {
			return walkDirHandler(ctx, rootName, path, d, err)
		})
		if err != nil {
			logging.ExitErr(logg, err)
			return err
		}
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dBLoopupByPathWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.DBLooklup", nil)
	if ctx.Database == nil {
		err := fmt.Errorf("dBLoopupByPathWorker: DB is nil")
		logging.ExitErr(logg, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		logg := logging.Enter(ctx.Ctx, "pipeline.DBLooklup.job", map[string]any{
			"path": job.RealPath,
		})
		image, err := dao.GetImageByPath(ctx.Database, ctx.Ctx, job.RootName, job.Path, job.Filename)
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
			logging.ExitErr(logg, err)
			return err
		}
		if ctx.Force {
			job.IsDirty = true
			job.DirtyReason = dirtyForced
		}
		logging.Exit(logg, "ok", nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dirtyCheckWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.dirtyChecker", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			logging.ExitErr(logg, ctx.Ctx.Err())
			return ctx.Ctx.Err()
		default:
		}
		logg := logging.Enter(ctx.Ctx, "pipeline.dirtyChecker.job", map[string]any{
			"path": job.RealPath,
		})

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
		logging.Exit(logg, "ok", nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}
func metadataReaderWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.metadataReader", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	var panoramaCheck ruleengine.CompiledFilter = nil
	if ctx.Panorama != nil {
		var err error
		panoramaCheck, err = ruleengine.CompileGroupFilter(*ctx.Panorama)
		if err != nil {
			logging.ErrorContinue(logg, err, map[string]any{"filter": "panorama"})
			panoramaCheck = nil
		}
	}
	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.metaReader.job", map[string]any{
			"path":  job.RealPath,
			"dirty": job.IsDirty,
		})
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logg, err)
			return err
		default:
		}
		log := "nop"
		if job.IsDirty {
			log = "ok"
			path := []string{job.RealPath}
			if job.MetadataFile != "" {
				path = append(path, job.MetadataFile)
			}
			metadata, err := metadata.ExtractMetadata(ctx.Ctx, path...)
			if err != nil {
				logging.ExitErr(logg, err)
				continue
			}
			job.Metadata = metadata
			if panoramaCheck != nil {
				facts := createImageFact(job)
				panorama := panoramaCheck(facts)
				job.Panorama = panorama
			}
		}
		logging.Exit(logg, log, nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}
func filterWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.filterWorker", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	type Dirkey struct {
		Root string
		Path string
	}
	filters := map[Dirkey]ruleengine.CompiledFilter{}
	for _, f := range ctx.Filters {
		key := Dirkey{
			Root: f.Root,
			Path: f.Path,
		}
		filterGrp := f.Filters
		cf, err := ruleengine.CompileGroupFilter(filterGrp)
		if err != nil {
			logging.ErrorContinue(logg, err, map[string]any{
				"filter": key,
			})
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
		logg := logging.Enter(ctx.Ctx, "pipeline.filterWorker.job", map[string]any{
			"path": job.RealPath,
		})
		facts := createImageFact(job)
		match := true
		notKey := Dirkey{}
		for key, filter := range filters {
			root := key.Root
			path := key.Path
			if job.RootName == root && strings.HasPrefix(job.Path, path) {
				match = filter(facts)
				if !match {
					notKey = key
					break
				}
			}
		}
		if !match {
			log.Logger.Info().Str("path", job.RealPath).Str("rule path", notKey.Path).Str("rule root", notKey.Root).Msg("Filtered out")
			log.Logger.Debug().Str("path", job.RealPath).Msg("filterWorker finished a job")
			continue
		}
		logging.Exit(logg, "ok", nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func aclWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "image.sync.aclWorker", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	ACLRules := ACLRules{}
	for i, acl := range ctx.ACLRules {
		aclRule := ACLRule{
			Role: acl.Role,
			User: acl.User,
		}
		// TODO: db lookup for userId by username
		rules := []ruleengine.CompiledFilter{}
		for j, rawRule := range acl.Rules {
			r, err := ruleengine.CompileGroupFilter(*rawRule)
			if err != nil {
				logging.ErrorContinue(logg, err, map[string]any{"aclRule": i, "ruleSeq": j})
				continue
			}
			rules = append(rules, r)
		}
		aclRule.Rules = rules
		ACLRules = append(ACLRules, aclRule)
	}

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		logg := logging.Enter(ctx.Ctx, "pipeline.aclWorker.job", map[string]any{
			"path": job.RealPath,
		})
		if job.IsDirty || ctx.ACLOverride {
			facts := createImageFact(job)
			match := false
			for i, aclRule := range ACLRules {
				for j, rule := range aclRule.Rules {
					match := rule(facts)
					logging.Inside(logg, map[string]any{
						"acl_rule": i,
						"rule":     j,
						"result":   match,
					}, "")
					if match {
						if !job.IsDirty {
							job.IsDirty = true
							job.DirtyReason = "ACL setting"
						}
						job.ACLRole = mapper.MapACL(aclRule.Role, aclRule.UserId)
						job.ACLUser = aclRule.UserId
						lg := log.Logger.Info()
						lg.Str("path", job.RealPath).Int("ACL_rule", i).
							Int("rule", j).Str("acl", string(aclRule.Role))
						logging.StrIf(lg, "user", aclRule.User)
						lg.Msg("Set ACL")
						logging.Info("pipeline.aclWorker.job", "ACL check", "Ok", "", map[string]any{
							"acl_rule": i,
							"rule":     j,
							"acl_role": aclRule.Role,
							"acl_user": aclRule.User,
						})
						break
					}
				}
				if match {
					break
				}
			}
			if !match {
				log.Logger.Info().Str("path", job.RealPath).Msg("No ACL rule found")
			}
		} else {
			log.Logger.Info().Str("path", job.RealPath).Msg("Not handled by ACL Worker")
		}
		logging.Exit(logg, "ok", nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dBUpdateWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.dbUpdater", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("dBUpdateWorker: In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.dbUpdater.job", map[string]any{
			"path":  job.RealPath,
			"dirty": job.IsDirty,
		})
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logg, err)
			return err
		default:
		}

		if job.IsDirty {
			job.DBImage.Root = job.RootName
			job.DBImage.Path = job.Path
			job.DBImage.Filename = job.Filename
			job.DBImage.FileSize = uint64(job.Info.Size())
			job.DBImage.MTime = job.Info.ModTime()
			job.DBImage.Ext = job.Ext
			job.DBImage.FileHash = job.FileHash
			job.DBImage.MetaHash = job.FileMetadataHash
			job.DBImage.LastSeenSync = &ctx.SyncId
			if job.Panorama {
				job.DBImage.Panorama = 1
			} else {
				job.DBImage.Panorama = 0
			}
			if job.ACLRole != nil {
				job.DBImage.ACLScope = dbo.ACLScope(*job.ACLRole)
			}
			if job.ACLUser != nil {
				job.DBImage.ACLUserID = job.ACLUser
			}
			mapper.UpdateImageMetadata(job.DBImage, job.Metadata)
			updateID, err := dao.CreateOrUpdateImage(ctx.Database, ctx.Ctx, job.DBImage)
			if err != nil {
				logging.ExitErrParams(logg, err, map[string]any{"is_dirty": job.IsDirty})
				continue
			} else {
				job.DBImage.ID = &updateID
			}
		}
		err := dao.UpdateImageSyncId(ctx.Database, ctx.Ctx, *job.DBImage.ID, ctx.SyncId)
		if err != nil {
			logging.ExitErrParams(logg, err, map[string]any{"is_dirty": job.IsDirty})
			continue
		}
		logging.Exit(logg, "ok", nil)
		ctx.Out <- job
	}
	logging.Exit(logg, "ok", nil)
	return nil
}
