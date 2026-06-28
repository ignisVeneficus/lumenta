package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/exif"
	"github.com/ignisVeneficus/lumenta/metadata"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/utils"
)

func walkDirHandler(ctx *PipelineContext, rootName, root string, excludedDirNames, excludedPaths map[string]struct{}, realPath string, d fs.DirEntry, err error, logScope logging.LogScope, localCtx context.Context) error {

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
			logging.Info(logScope, "Skipped by name", map[string]any{
				"real_path": realPath,
				"name":      root,
			})
			return filepath.SkipDir

		}
		if _, ok := excludedPaths[fullPath]; ok {
			logging.Info(logScope, "Skipped by path", map[string]any{
				"real_path": realPath,
				"full_path": fullPath,
			})
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
	logScope, jobCtx := logging.Enter(localCtx, "pipeline/job/run", realPath, map[string]any{
		"root":          root,
		"root_name":     rootName,
		"real_path":     realPath,
		"local_path":    path,
		"filename":      filename,
		"ext":           ext,
		"metadata_file": metaFile,
	})
	ws := time.Now()
	select {
	case ctx.Out <- WorkItem{
		Ctx:          jobCtx,
		LogScope:     logScope,
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
	logging.Debug(logScope, "pipeline insert", map[string]any{
		"wait_insert": time.Since(ws),
	})

	return nil
}

func fSWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/fs_walker/run", nil, nil)

	for rootName, rootConfig := range ctx.RootPath {
		logScope, logCtx := logging.Enter(ctx.Ctx, "sync/pipeline/fs_walker/root", rootName, nil)

		excludedDirNames := make(map[string]struct{})
		for _, n := range rootConfig.ExcludedDirs {
			excludedDirNames[n] = struct{}{}
		}
		excludedPath := make(map[string]struct{})
		for _, n := range rootConfig.ExcludedPath {
			excludedPath[n] = struct{}{}
		}

		err := filepath.WalkDir(rootConfig.Root, func(path string, d fs.DirEntry, err error) error {
			return walkDirHandler(ctx, rootName, rootConfig.Root, excludedDirNames, excludedPath, path, d, err, logScope, logCtx)
		})
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", nil)
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}
func convertAlbums(albums []dbo.AlbumID, ctx *PipelineContext) ruleengine.AlbumsStruct {
	ret := make(ruleengine.AlbumsStruct)
	for _, album := range albums {
		as, ok := ctx.AlbumCtx.AlbumStructs[uint64(album)]
		if ok {
			ret[uint64(album)] = as
		}
	}
	return ret
}

func dBLoopupByPathWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/db_lookup/run/inside", nil, nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/db_lookup", job.RealPath, map[string]any{
			"path": job.RealPath,
		})
		image, err := dao.GetImageByPath(ctx.Database, c, job.RootName, job.Path, job.Filename, job.Ext)
		switch {
		case err == nil:
			job.DBImage = &image
			setJobFromImage(&job)
			albums, err := dao.QueryAlbumsIDByImageID(ctx.Database, c, *job.DBImage.ID)
			if err != nil {
				logging.ExitErr(logScope, err)
				return err
			}
			job.Albums = convertAlbums(albums, ctx)
		case errors.Is(err, dao.ErrDataNotFound):
			filtered, err := dao.GetFilteredByPath(ctx.Database, c, job.RootName, job.Path, job.Filename, job.Ext)
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
				logging.ExitErr(logScope, err)
				return err
			}
		default:
			logging.ExitErr(logScope, err)
			return err
		}
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func hashWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/hash/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			logging.ExitErr(logScope, ctx.Ctx.Err())
			return ctx.Ctx.Err()
		default:
		}
		logScope, _ := logging.Enter(job.Ctx, "pipeline/job/run/hash", job.RealPath, map[string]any{
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

		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func dirtyCheckWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/dirty_check/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		logScope, _ := logging.Enter(job.Ctx, "pipeline/job/run/dirty_check", job.RealPath, map[string]any{
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
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"is_dirty":     job.IsDirty,
			"dirty_reason": job.DirtyReason,
			"wait_insert":  time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}
func metadataReaderWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/metadat_reader/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	var panoramaCheck ruleengine.CompiledGroupFilter = nil
	if ctx.Panorama != nil {
		var err error
		panoramaCheck, err = ruleengine.CompileGroupFilter(*ctx.Panorama, "Panorama")
		if err != nil {
			logging.ErrorContinue(logScope, err, map[string]any{"filter": "panorama"})
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
		logging.ExitErr(logScope, err)
		return err
	}

	defer exiftool.Close()

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/metadat_reader", job.RealPath, map[string]any{
			"path": job.RealPath,
		})
		log := "nop"
		if job.IsDirty {
			log = "ok"
			path := []metadata.Path{
				{
					Path:     job.RealPath,
					PathType: metadata.PathTypeImage,
				},
			}
			if job.MetadataFile != "" {
				path = append(path, metadata.Path{
					Path:     job.MetadataFile,
					PathType: metadata.PathTypeSidecar,
				},
				)
			}

			metadata, err := metadata.ExtractMetadata(exiftool, c, path...)
			if err != nil {
				logging.ExitErr(logScope, err)
				SaveResultError(ctx, job, c)
				_ = exiftool.Close()
				err = createTool()
				if err != nil {
					logging.ExitErr(logScope, err)
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
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, log, map[string]any{
			"wait_insert": time.Since(ws),
		})
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func filterWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/import_filter/run/inside", nil, nil)

	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
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
			logging.ErrorContinue(logScope, err, map[string]any{
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
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/import_filter", job.RealPath, map[string]any{
			"path":     job.RealPath,
			"metadata": job.Metadata,
		})

		facts := createImageFact(job)
		logging.Debug(logScope, "imagefacts", map[string]any{"facts": facts, "job": job})
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
			SaveResultSkip(ctx, job, c)
			filtered := ctx.FilterOut
			ws := time.Now()
			if filtered != nil {
				select {
				case filtered <- job:
				case <-ctx.Ctx.Done():
					err := ctx.Ctx.Err()
					logging.ExitErr(logScope, err)
					return err
				}
			}
			logging.Exit(logScope, "skipped", map[string]any{
				"source":      job.Source,
				"wait_insert": time.Since(ws),
			})
			continue
		}
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func aclWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/acl_rules/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
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
				logging.ErrorContinue(logScope, err, map[string]any{"aclRule": i, "ruleSeq": j})
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
		logScope, _ := logging.Enter(job.Ctx, "pipeline/job/run/acl_rules", job.RealPath, map[string]any{
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
					logging.Trace(logScope, "acl_rule", map[string]any{
						"acl_rule": i,
						"rule":     j,
						"result":   match,
					})
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
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func dbImageWriterWorker(ctx *PipelineContext, tagCache *TagCache) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/image_writer/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/acl_rules", job.RealPath, map[string]any{
			"path": job.RealPath,
		})

		if job.IsDirty {
			getDBOImageFromJob(job, ctx.SyncId, ctx.Force)
			updateID, err := dao.CreateOrUpdateImage(ctx.Database, ctx.Ctx, job.DBImage)
			if err != nil {
				logging.ExitErrParams(logScope, err, map[string]any{"is_dirty": job.IsDirty})
				SaveResultError(ctx, job, c)
				continue
			}
			tagSet := make(map[dbo.TagID]struct{})
			tags := job.Metadata.GetTags()
			for _, t := range tags {
				var tagIDs []dbo.TagID
				tagIDs, err = tagCache.Resolve(ctx.Database, c, t, "Digikam")
				if err != nil {
					logging.ExitErrParams(logScope, err, map[string]any{"is_dirty": job.IsDirty})
					SaveResultError(ctx, job, c)
					break
				}
				for _, id := range tagIDs {
					tagSet[id] = struct{}{}
				}
			}
			if err != nil {
				continue
			}
			tagIDs := make([]dbo.TagID, 0, len(tagSet))
			for id := range tagSet {
				tagIDs = append(tagIDs, id)
			}
			err = dao.BindImageTags(ctx.Database, c, updateID, tagIDs)
			if err != nil {
				logging.ExitErrParams(logScope, err, map[string]any{"is_dirty": job.IsDirty})
				SaveResultError(ctx, job, c)
				continue
			}
			job.DBImage.ID = &updateID

		}
		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"is_dirty":     job.IsDirty,
			"dirty_reason": job.DirtyReason,
			"source":       job.Source,
			"wait_insert":  time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func albumInsertionWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/album_rules/run/inside", nil, nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	albumsRules := ctx.AlbumCtx.Rules
	ruleCtx := ruleengine.RuleContext{
		NameMap: ctx.AlbumCtx.NameMap,
	}

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/album_rules", job.RealPath, map[string]any{
			"path": job.RealPath,
		})
		if job.DBImage != nil {
			//			facts := createImageFactDb(job)
			facts := createImageFact(job)
			for _, ar := range albumsRules {
				if ar.Rule == nil {
					logging.Trace(logScope, "empty rule", map[string]any{
						"album_id":   ar.ID,
						"album_name": ar.Name,
					})
					continue
				}
				rctx := ruleCtx
				rctx.RefAlbum = &ar.ID
				match, ruleResult := ar.Rule(facts, &rctx)
				_, found := job.Albums[ar.ID]
				job.RuleResults.AddResult(ruleengine.EvaluationAlbum, ruleResult)
				if found && !match {
					err := dao.BreakAlbumImage(ctx.Database, c, dbo.AlbumID(ar.ID), *job.DBImage.ID)
					if err != nil {
						logging.ErrorContinue(logScope, err, map[string]any{
							"album_id": ar.ID,
							"image_id": job.DBImage.ID,
						})
					}
					delete(facts.Albums, ar.ID)
				}
				if !found && match {
					err := dao.BindAlbumImage(ctx.Database, c, dbo.AlbumID(ar.ID), *job.DBImage.ID, nil)
					if err != nil {
						logging.ErrorContinue(logScope, err, map[string]any{
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

		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func resultSaverWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/result_saver/run/inside", nil, nil)
	if ctx.Database == nil {
		err := fmt.Errorf("DB is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}

	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/result_saver", job.RealPath, map[string]any{
			"path":  job.RealPath,
			"dirty": job.IsDirty,
		})
		err := dao.UpdateImageSyncID(ctx.Database, c, *job.DBImage.ID, ctx.SyncId)
		if err != nil {
			logging.ExitErrParams(logScope, err, map[string]any{"is_dirty": job.IsDirty})
			SaveResultError(ctx, job, c)
			continue
		}
		err = SaveResultSucess(ctx, job, c)
		if err != nil {
			logging.ExitErrParams(logScope, err, map[string]any{"is_dirty": job.IsDirty})
			SaveResultError(ctx, job, c)
			continue
		}

		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"wait_insert": time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func dbFilteredWriterWorker(ctx *PipelineContext) error {
	logScope, _ := logging.Enter(ctx.Ctx, "sync/pipeline/filtered_writer/run/inside", nil, nil)
	if ctx.In == nil || ctx.Out == nil {
		err := fmt.Errorf("In/Out channel is nil")
		logging.ExitErr(logScope, err)
		return err
	}
	for job := range ctx.In {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		default:
		}
		logScope, c := logging.Enter(job.Ctx, "pipeline/job/run/filtered_writer", job.RealPath, map[string]any{
			"path":  job.RealPath,
			"dirty": job.IsDirty,
		})
		filtered := getDBOFilteredFromJob(job, ctx.SyncId)
		err := dao.CreateOrUpdateFiltered(ctx.Database, c, &filtered)
		if err != nil {
			logging.ExitErr(logScope, err)
			SaveResultError(ctx, job, c)
			continue
		}

		ws := time.Now()
		select {
		case ctx.Out <- job:
		case <-ctx.Ctx.Done():
			err := ctx.Ctx.Err()
			logging.ExitErr(logScope, err)
			return err
		}
		logging.Exit(logScope, "ok", map[string]any{
			"is_dirty":     job.IsDirty,
			"dirty_reason": job.DirtyReason,
			"wait_insert":  time.Since(ws),
		})

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}
