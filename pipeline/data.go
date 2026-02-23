package pipeline

import (
	"context"
	"database/sql"
	"os"
	"sort"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	fileConfig "github.com/ignisVeneficus/lumenta/config/filesystem"
	syncConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog"
)

type dirtyReason string

const (
	dirtyNewfile         dirtyReason = "new_file"
	dirtyHashChg         dirtyReason = "hash_changed"
	dirtyMetadataHashChg dirtyReason = "metadata_hash_changed"
	dirtySizeChg         dirtyReason = "size_changed"
	dirtyTimeChg         dirtyReason = "mtime_changed"
	dirtyForced          dirtyReason = "forced_refresh"
)

// WorkItem is a pipeline state object that flows through the sync pipeline.
// Rule: each worker may ONLY read existing fields and append new information.
// Fields must never be cleared or overwritten with weaker data.
// The WorkItem accumulates all knowledge about a file from filesystem walk
// to database persistence in a single, mutable envelope.
type WorkItem struct {

	// =========================================================
	// FILESYSTEM (walk)
	// =========================================================

	RootName     string      // name of the root, defined in config
	RootPath     string      // configured sync root
	Path         string      // path relative to RootPath without filename
	RealPath     string      // absolute / canonical filesystem path
	Filename     string      // base filename
	MetadataFile string      // metadata filename
	Ext          string      // normalized extension (without dot)
	Info         os.FileInfo // filesystem stat info

	// =========================================================
	// DATABASE PRECHECK (path-based lookup)
	// =========================================================

	// Exists in db => DBImage.ID not null
	DBImage *dbo.Image // persisted DB object (or nil if skipped)

	// =========================================================
	// DIRTY CHECK / CHANGE DETECTION
	// =========================================================

	IsDirty     bool        // true if file must be re-processed
	DirtyReason dirtyReason // human-readable reason (debug / metrics)

	// =========================================================
	// CONTENT HASH (optional, policy-driven)
	// =========================================================

	FileHash         string // computed content hash
	FileMetadataHash string // computed content hash

	// =========================================================
	// ACL
	// =========================================================

	ACLRole *dbo.ACLScope
	ACLUser *uint64

	// =========================================================
	// METADATA EXTRACTION (EXIF / XMP / IPTC)
	// =========================================================

	Metadata data.Metadata
	Panorama bool

	// =========================================================
	// ERROR / DIAGNOSTICS
	// =========================================================

	Err error // non-fatal processing error (does not stop pipeline)
}

func (w *WorkItem) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {

	if level <= zerolog.DebugLevel {

		e.Str("root_path", w.RootPath).
			Str("path", w.Path).
			Str("real_path", w.RealPath).
			Str("filename", w.Filename).
			Str("ext", w.Ext)

		if w.Info != nil {
			e.Int64("fs_size", w.Info.Size()).
				Time("fs_mtime", w.Info.ModTime())
		}

		e.Bool("exists_in_db", (w.DBImage != nil && w.DBImage.ID != nil)).
			Bool("is_dirty", w.IsDirty)
		if w.DirtyReason != "" {
			e.Str("dirty_reason", string(w.DirtyReason))
		}
		if w.DBImage != nil && w.DBImage.ID != nil {
			logging.Uint64If(e, "db_id", w.DBImage.ID)
		}
	}

	if level == zerolog.TraceLevel {

		e.Str("file_hash", w.FileHash)

		// Nested objects – delegated logging

		logging.ObjectIf(e, "metadata", logging.WithLevel(level, &w.Metadata), false)
		logging.ObjectIf(e, "db_image", logging.WithLevel(level, w.DBImage), false)
		if w.Err != nil {
			e.Err(w.Err)
		}
	}
}

type PipelineContext struct {
	// =========================================================
	// Execution
	// =========================================================

	Ctx    context.Context
	Cancel context.CancelFunc

	// =========================================================
	// Configuration (read-only)
	// =========================================================

	RootPath   fileConfig.RootConfigs
	AllowedExt map[string]struct{}

	Database    *sql.DB
	Metadata    *syncConfig.MetadataConfig
	Filters     []syncConfig.PathFilterConfig
	Panorama    *ruleengine.RuleGroup
	ACLRules    syncConfig.ACLRules
	ACLOverride bool

	SyncId uint64
	Force  bool

	// =========================================================
	// Channels (may be nil)
	// =========================================================

	In  <-chan WorkItem
	Out chan<- WorkItem
}

type ACLRules []ACLRule

type ACLRule struct {
	Role   authData.ACLRole
	User   *string
	UserId *uint64
	Rules  []ruleengine.CompiledFilter
}

func (pc *PipelineContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {

	if level <= zerolog.DebugLevel {

		e.Bool("has_in", pc.In != nil).
			Bool("has_out", pc.Out != nil)

		//FIXME: add the originals to debug
		//e.Str("root_path", pc.RootPath)

		if len(pc.AllowedExt) > 0 {
			exts := make([]string, 0, len(pc.AllowedExt))
			for ext := range pc.AllowedExt {
				exts = append(exts, ext)
			}
			sort.Strings(exts)
			e.Strs("allowed_ext", exts)
		}
	}
}

func createImageFact(job WorkItem) ruleengine.ImageFacts {
	rating := 0
	if job.Metadata.GetRating() != nil {
		rating = int(*job.Metadata.GetRating())
	}
	return ruleengine.ImageFacts{
		Path:     job.Path,
		Filename: job.Filename,
		Ext:      job.Ext,
		TakenAt:  job.Metadata.GetTakenAt(),
		Rating:   &rating,
		Tags:     job.Metadata.GetTags(),
		Width:    job.Metadata.GetWidth(),
		Height:   job.Metadata.GetHeight(),
	}

}
