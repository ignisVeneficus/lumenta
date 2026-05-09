package pipeline

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/exif"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/metadata"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/rs/zerolog/log"
)

func walkDirHandler(ctx *PipelineContext, rootName, root string, excludedDirNames, excludedPaths map[string]struct{}, realPath string, d fs.DirEntry, err error) error {

	if err != nil {
		return err
	}

	select {
	case <-ctx.Ctx.Done():
		return ctx.Ctx.Err()
	default:
	}

	realPath = filepath.ToSlash(realPath)

	rel, err := filepath.Rel(root, realPath)
	rel = filepath.ToSlash(rel)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return nil
	}
	fullPath := rel
	if fullPath == "." {
		fullPath = ""
	}

	if d.IsDir() {
		name := d.Name()
		if _, ok := excludedDirNames[name]; ok {
			log.Logger.Info().Str("path", realPath).Str("name", rootName).Msg("Skipped for sync")
			return filepath.SkipDir

		}
		if _, ok := excludedPaths[fullPath]; ok {
			log.Logger.Info().Str("path", realPath).Str("name", rootName).Msg("Skipped for sync")
			return filepath.SkipDir
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

	if ctx.Out == nil {
		return nil
	}

	select {
	case ctx.Out <- WorkItem{
		RootPath:     root,
		RootName:     rootName,
		Path:         path,
		RealPath:     realPath,
		MetadataFile: metaFile,
		Ext:          normalisedExt,
		Filename:     filename,
		Info:         info,
	}:
	case <-ctx.Ctx.Done():
		return ctx.Ctx.Err()
	}

	return nil
}

func fSWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.fsWalker", nil)

	for rootName, rootConfig := range ctx.RootPath {

		excludedDirNames := make(map[string]struct{})
		for _, n := range rootConfig.ExcludedDirs {
			excludedDirNames[n] = struct{}{}
		}
		excludedPath := make(map[string]struct{})
		for _, n := range rootConfig.ExcludedPath {
			excludedPath[n] = struct{}{}
		}

		err := filepath.WalkDir(rootConfig.Root, func(path string, d fs.DirEntry, err error) error {
			return walkDirHandler(ctx, rootName, rootConfig.Root, excludedDirNames, excludedPath, path, d, err)
		})
		if err != nil {
			logging.ExitErr(logg, err)
			return err
		}
	}
	logging.Exit(logg, "ok", nil)
	return nil
}
func convertAlbums(albums []uint64, ctx *PipelineContext) ruleengine.AlbumsStruct {
	ret := make(ruleengine.AlbumsStruct)
	for _, album := range albums {
		as, ok := ctx.AlbumCtx.AlbumStructs[album]
		if ok {
			ret[album] = as
		}
	}
	return ret
}

func dBLoopupByPathWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.DBLooklup", nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
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
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.DBLooklup.job", map[string]any{
			"path": job.RealPath,
		})
		image, err := dao.GetImageByPath(ctx.Database, ctx.Ctx, job.RootName, job.Path, job.Filename, job.Ext)
		switch {
		case err == nil:
			job.DBImage = &image
			setJobFromImage(&job)
			albums, err := dao.QueryAlbumIDByImageID(ctx.Database, ctx.Ctx, *job.DBImage.ID)
			if err != nil {
				logging.ExitErr(logg, err)
				return err
			}
			job.Albums = convertAlbums(albums, ctx)
		case errors.Is(err, dao.ErrDataNotFound):
			filtered, err := dao.GetFilteredByPath(ctx.Database, ctx.Ctx, job.RootName, job.Path, job.Filename, job.Ext)
			switch {
			case err == nil:
				setJobFromFiltered(&job, filtered)
			case errors.Is(err, dao.ErrDataNotFound):
				job.Source = SourceFS
				job.DBImage = &dbo.Image{
					FocusMode: dbo.ImageFocusModeAuto,
					ACLLevel:  dbo.DBACLLevelPublic,
				}
			default:
				logging.ExitErr(logg, err)
				return err
			}
		default:
			logging.ExitErr(logg, err)
			return err
		}
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func hashWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.hash", nil)
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
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.hash.job", map[string]any{
			"path": job.RealPath,
		})

		fileHash, err := utils.ComputeFileHash(job.RealPath)
		if err != nil {
			return nil
		}
		metaHash, err := utils.ComputeFileHash(job.MetadataFile)
		if err != nil {
			if os.IsNotExist(err) {
				metaHash = ""
				job.MetadataFile = ""
			} else {
				return err
			}
		}
		job.FileHash = fileHash
		job.FileMetadataHash = metaHash

		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dirtyCheckWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.dirtyChecker", nil)
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
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.dirtyChecker.job", map[string]any{
			"path": job.RealPath,
		})

		if job.Source != SourceFS {
			for {
				if job.FileHash != job.CachedFileHash {
					job.IsDirty = true
					job.DirtyReason = data.DirtyHashChg
					break
				}
				if job.FileMetadataHash != job.CachedFileMetadataHash {
					job.IsDirty = true
					job.DirtyReason = data.DirtyMetadataHashChg
					break
				}
				if job.CachedSize != uint64(job.Info.Size()) {
					job.IsDirty = true
					job.DirtyReason = data.DirtySizeChg
					break
				}
				if !utils.SameTime(job.Info.ModTime(), job.CachedTime) {
					job.IsDirty = true
					job.DirtyReason = data.DirtyTimeChg
					break
				}
				break
			}
		} else {
			job.IsDirty = true
			job.DirtyReason = data.DirtyNewfile
		}

		if ctx.Force {
			job.IsDirty = true
			job.DirtyReason = data.DirtyForced
		}

		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}
func metadataReaderWorker(ctx *PipelineContext) error {
	loggWorker := logging.Enter(ctx.Ctx, "pipeline.worker.metadataReader", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(loggWorker, err)
		return err
	}
	var panoramaCheck ruleengine.CompiledGroupFilter = nil
	if ctx.Panorama != nil {
		var err error
		panoramaCheck, err = ruleengine.CompileGroupFilter(*ctx.Panorama, "Panorama")
		if err != nil {
			logging.ErrorContinue(loggWorker, err, map[string]any{"filter": "panorama"})
			panoramaCheck = nil
		}
	}

	var exiftool *exif.PersistentExiftool
	var err error

	createTool := func() error {

		if exiftool != nil {
			_ = exiftool.Close()
		}

		exiftool, err = exif.NewPersistentExiftool(
			ctx.Ctx,
			ctx.ExifToolConfig.Path,
			ctx.ExifToolConfig.Timeout,
		)

		return err
	}

	err = createTool()
	if err != nil {
		logging.ExitErr(loggWorker, err)
		return err
	}

	defer exiftool.Close()

	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.metaReader.job", map[string]any{
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
			metadata, err := metadata.ExtractMetadata(exiftool, ctx.Ctx, path...)
			if err != nil {
				logging.ExitErr(logg, err)
				SaveResultError(ctx, job)
				_ = exiftool.Close()
				err = createTool()
				if err != nil {
					logging.ExitErr(loggWorker, err)
					return err
				}
				continue
			}
			job.Metadata = metadata
			if panoramaCheck != nil {
				facts := createImageFact(job)
				panorama, panoramafilterResult := panoramaCheck(facts, nil)
				job.RuleResults.AddResult(ruleengine.EvaluationPanorama, panoramafilterResult)
				job.Panorama = panorama
			}
		}
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, log, nil)

	}
	logging.Exit(loggWorker, "ok", nil)
	return nil
}
func filterWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.filter", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	if ctx.FilterOut != nil {
		defer ctx.WG.Done()
	}
	type DirKey struct {
		Root string
		Path string
	}
	filters := map[DirKey]ruleengine.CompiledGroupFilter{}
	for _, f := range ctx.Filters {
		key := DirKey{
			Root: f.Root,
			Path: f.Path,
		}
		name := fmt.Sprintf("PathFilter:%s:%s", f.Root, f.Path)
		filterGrp := f.Filters
		cf, err := ruleengine.CompileGroupFilter(filterGrp, name)
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
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.filter.job", map[string]any{
			"path":     job.RealPath,
			"metadata": job.Metadata,
		})
		facts := createImageFact(job)
		logging.Inside(logg, map[string]any{"facts": facts, "job": job}, "Imagefacts")
		match := true
		for key, filter := range filters {
			root := key.Root
			path := key.Path
			if job.RootName == root && strings.HasPrefix(job.Path, path) {
				res, pathFilterResult := filter(facts, nil)
				pathFilterResult.Params = append(pathFilterResult.Params,
					ruleengine.CreateRuleParamString("root", root),
					ruleengine.CreateRuleParamString("path", path),
				)
				job.RuleResults.AddResult(ruleengine.EvaluationFilesystem, pathFilterResult)
				if !res {
					match = false
					break
				}
			}
		}
		if !match {
			SaveResultSkip(ctx, job)
			logging.Exit(logg, "skipped", nil)
			filtered := ctx.FilterOut
			if filtered != nil {
				select {
				case filtered <- job:
				case <-ctx.Ctx.Done():
					return ctx.Ctx.Err()
				}
			}
			continue
		}
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func aclWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.acl", nil)
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
		rules := []ruleengine.CompiledGroupFilter{}
		for j, rawRule := range acl.Rules {
			name := fmt.Sprintf("ACL Filter:%s:%s:%d", acl.Role, "", j)
			r, err := ruleengine.CompileGroupFilter(*rawRule, name)
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
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.acl.job", map[string]any{
			"path": job.RealPath,
		})
		if job.IsDirty || ctx.ACLOverride {
			facts := createImageFact(job)
			match := false
			for i, aclRule := range ACLRules {
				for j, rule := range aclRule.Rules {
					var ruleResult ruleengine.GroupRuleResult
					match, ruleResult = rule(facts, nil)
					job.RuleResults.AddResult(ruleengine.EvaluationACL, ruleResult)
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
						job.ACLLevel = aclRule.GetACLLevel()
						job.ACLUser = 0
						if aclRule.UserId != nil {
							job.ACLUser = *aclRule.UserId
						}
						break
					}
				}
				if match {
					break
				}
			}
		}
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dbImageWriterWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.imageWriter", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.imageWriter.job", map[string]any{
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
			getDBOImageFromJob(job, ctx.SyncId)
			updateID, err := dao.CreateOrUpdateImage(ctx.Database, ctx.Ctx, job.DBImage)
			if err != nil {
				logging.ExitErrParams(logg, err, map[string]any{"is_dirty": job.IsDirty})
				SaveResultError(ctx, job)
				continue
			} else {
				job.DBImage.ID = &updateID
			}
		}
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func albumInsertionWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.albumInserter", nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
		logging.ExitErr(logg, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	albumsRules := ctx.AlbumCtx.Rules
	ruleCtx := ruleengine.RuleContext{
		NameMap: ctx.AlbumCtx.NameMap,
	}

	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.albumInserter.job", map[string]any{
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
		if job.DBImage != nil {
			//			facts := createImageFactDb(job)
			facts := createImageFact(job)
			for _, ar := range albumsRules {
				if ar.Rule == nil {
					logging.Inside(logg, map[string]any{
						"album_id":   ar.ID,
						"album_name": ar.Name,
					}, "empty rule")
					continue
				}
				rctx := ruleCtx
				rctx.RefAlbum = &ar.ID
				match, ruleResult := ar.Rule(facts, &rctx)
				_, found := job.Albums[ar.ID]
				job.RuleResults.AddResult(ruleengine.EvaluationAlbum, ruleResult)
				if found && !match {
					err := dao.BreakAlbumImage(ctx.Database, ctx.Ctx, ar.ID, *job.DBImage.ID)
					if err != nil {
						logging.ErrorContinue(logg, err, map[string]any{
							"album_id": ar.ID,
							"image_id": job.DBImage.ID,
						})
					}
					delete(facts.Albums, ar.ID)
				}
				if !found && match {
					err := dao.BindAlbumImage(ctx.Database, ctx.Ctx, ar.ID, *job.DBImage.ID, nil)
					if err != nil {
						logging.ErrorContinue(logg, err, map[string]any{
							"album_id": ar.ID,
							"image_id": job.DBImage.ID,
						})
					}
					as, ok := ctx.AlbumCtx.AlbumStructs[ar.ID]
					if ok {
						if facts.Albums == nil {
							facts.Albums = make(ruleengine.AlbumsStruct)
						}
						facts.Albums[ar.ID] = as
					}
				}

			}
		}

		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func resultSaverWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.resultSaver", nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
		logging.ExitErr(logg, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}

	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.resultSaver.job", map[string]any{
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
		err := dao.UpdateImageSyncId(ctx.Database, ctx.Ctx, *job.DBImage.ID, ctx.SyncId)
		if err != nil {
			logging.ExitErrParams(logg, err, map[string]any{"is_dirty": job.IsDirty})
			SaveResultError(ctx, job)
			continue
		}
		err = SaveResultSucess(ctx, job)
		if err != nil {
			logging.ExitErrParams(logg, err, map[string]any{"is_dirty": job.IsDirty})
			SaveResultError(ctx, job)
			continue
		}

		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func dbFilteredWriterWorker(ctx *PipelineContext) error {
	logg := logging.Enter(ctx.Ctx, "pipeline.worker.filteredWriter", nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logg, err)
		return err
	}
	for job := range ctx.In {
		logg := logging.Enter(ctx.Ctx, "pipeline.worker.filteredWriter.job", map[string]any{
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
		filtered := getDBOFilteredFromJob(job, ctx.SyncId)
		err := dao.CreateOrUpdateFiltered(ctx.Database, ctx.Ctx, &filtered)
		if err != nil {
			logging.ExitErr(logg, err)
			SaveResultError(ctx, job)
			continue
		}

		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		}
		logging.Exit(logg, "ok", nil)

	}
	logging.Exit(logg, "ok", nil)
	return nil
}
