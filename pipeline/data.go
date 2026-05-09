package pipeline

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"sync"
	"time"

	fileConfig "github.com/ignisVeneficus/lumenta/config/filesystem"
	syncConfig "github.com/ignisVeneficus/lumenta/config/sync"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/rs/zerolog"
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

	Source DataSource

	// Exists in db => DBImage.ID not null
	DBImage *dbo.Image // persisted DB object (or nil if skipped)
	Albums  ruleengine.AlbumsStruct

	// =========================================================
	// CACHED DATA (from one of table)
	// =========================================================
	CachedFileHash         string // computed content hash
	CachedFileMetadataHash string // computed content hash
	CachedSize             uint64
	CachedTime             time.Time

	// =========================================================
	// DIRTY CHECK / CHANGE DETECTION
	// =========================================================

	IsDirty     bool             // true if file must be re-processed
	DirtyReason data.DirtyReason // human-readable reason (debug / metrics)

	// =========================================================
	// CONTENT HASH (optional, policy-driven)
	// =========================================================

	FileHash         string // computed content hash
	FileMetadataHash string // computed content hash

	// =========================================================
	// ACL
	// =========================================================

	ACLLevel *dbo.DBACLLevel
	ACLUser  uint64

	// =========================================================
	// METADATA EXTRACTION (EXIF / XMP / IPTC)
	// =========================================================

	Metadata data.Metadata
	Panorama bool

	// =========================================================
	// RULE ENGINE RESULTS
	// =========================================================

	RuleResults ruleengine.RuleResults

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

type DataSource string

const (
	SourceImages   DataSource = "images"
	SourceFiltered DataSource = "filteredOut"
	SourceFS       DataSource = "fileesystem"
)

type PipelineContext struct {
	// =========================================================
	// Execution
	// =========================================================

	Ctx    context.Context
	Cancel context.CancelCauseFunc

	// =========================================================
	// Configuration (read-only)
	// =========================================================

	RootPath   fileConfig.RootConfigs
	AllowedExt map[string]struct{}

	Database       *sql.DB
	Metadata       *syncConfig.MetadataConfig
	Filters        []syncConfig.PathFilterConfig
	ExifToolConfig syncConfig.ExiftoolConfig
	Workers        map[syncConfig.StepName]syncConfig.StepConfig
	Panorama       *ruleengine.RuleGroup
	ACLRules       syncConfig.ACLRules
	ACLOverride    bool

	// =========================================================
	// Sync related data
	// =========================================================
	SyncId uint64
	Force  bool

	// =========================================================
	// Album struct
	// =========================================================
	AlbumCtx *AlbumContext

	// =========================================================
	// Channels (may be nil)
	// =========================================================

	In        <-chan WorkItem
	Out       chan<- WorkItem
	FilterOut chan<- WorkItem
	WG        *sync.WaitGroup
}

type ACLRules []ACLRule

type ACLRule struct {
	Role   dbo.ACLRole
	User   *string
	UserId *uint64
	Rules  []ruleengine.CompiledGroupFilter
}

func (a ACLRule) GetACLLevel() *dbo.DBACLLevel {
	v := dbo.DBACLLevelAdmin
	switch a.Role {
	case dbo.RoleGuest:
		v = dbo.DBACLLevelPublic
	case dbo.RoleUser:
		if a.User != nil {
			v = dbo.DBACLLevelUser
		} else {
			v = dbo.DBACLLevelAuthenticated
		}
	case dbo.RoleAdmin:
		v = dbo.DBACLLevelAdmin
	default:
		v = dbo.DBACLLevelAdmin
	}
	return &v
}

func (pc *PipelineContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {

	if level <= zerolog.DebugLevel {

		e.Bool("has_in", pc.In != nil).
			Bool("has_out", pc.Out != nil)

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
		Root:     job.RootName,
		Path:     job.Path,
		Filename: job.Filename,
		Ext:      job.Ext,
		TakenAt:  job.Metadata.GetTakenAt(),
		Rating:   &rating,
		Tags:     job.Metadata.GetTags(),
		Width:    job.Metadata.GetWidth(),
		Height:   job.Metadata.GetHeight(),
		Albums:   job.Albums,
	}

}

type AlbumRule struct {
	Name      string
	PathIDs   []uint64
	Rule      ruleengine.CompiledGroupFilter
	ID        uint64
	ParentID  *uint64
	Depth     int
	RankOrder []uint64
	Rank      uint64
	//	Path     string
}

type AlbumContext struct {
	NameMap      map[uint64]string
	AlbumStructs ruleengine.AlbumsStruct
	Rules        []*AlbumRule
}
