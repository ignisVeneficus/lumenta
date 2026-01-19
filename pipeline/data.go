package pipeline

import (
	"context"
	"database/sql"
	"os"
	"sort"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
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

	RootPath string      // configured sync root
	Path     string      // path relative to RootPath without filename
	RealPath string      // absolute / canonical filesystem path
	Filename string      // base filename
	Ext      string      // normalized extension (without dot)
	Info     os.FileInfo // filesystem stat info

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
	// METADATA EXTRACTION (EXIF / XMP / IPTC)
	// =========================================================

	Metadata data.Metadata

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

	RootPath   string
	PathConfig *[]config.PathFilterConfig
	AllowedExt map[string]struct{}

	Database *sql.DB
	Metadata *config.MetadataConfig
	Filters  []config.PathFilterConfig

	SyncId uint64

	// később:
	// DB ImageRepository
	// HashPolicy
	// FilterGroup
	// JobManager
	// Logger

	// =========================================================
	// Channels (may be nil)
	// =========================================================

	In  <-chan WorkItem
	Out chan<- WorkItem
}

func (pc *PipelineContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {

	if level <= zerolog.DebugLevel {

		e.Bool("has_in", pc.In != nil).
			Bool("has_out", pc.Out != nil)

		// Context cancellation state
		if pc.Ctx != nil {
			select {
			case <-pc.Ctx.Done():
				e.Bool("ctx_done", true)
			default:
				e.Bool("ctx_done", false)
			}
		}
		e.Str("root_path", pc.RootPath)

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
