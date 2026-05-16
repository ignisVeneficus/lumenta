package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/rs/zerolog"
)

// =========================================================
// FILTERED OUT IMAGES
// =========================================================
type FilteredID uint64
type FilteredOut struct {
	ID *FilteredID

	Root     string
	Path     string
	Filename string
	Ext      string

	FileSize uint64
	MTime    time.Time
	FileHash string
	MetaHash string

	ExifJSON json.RawMessage

	CreatedAt time.Time
	UpdatedAt time.Time

	LastSeenSync *SyncRunID
}

func (i *FilteredOut) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("path", i.Path).
			Str("filename", i.Filename).
			Str("ext", i.Ext).
			Str("root", i.Root)
		logging.Uint64If(e, "id", (*uint64)(i.ID))
		logging.Uint64If(e, "last sync", (*uint64)(i.LastSeenSync))

	}
	if level == zerolog.TraceLevel {
		e.Str("file_hash", i.FileHash).
			Str("meta_hash", i.MetaHash).
			Time("created_at", i.CreatedAt).
			Time("updated_at", i.UpdatedAt)

	}
}
