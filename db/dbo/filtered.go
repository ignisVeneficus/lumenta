package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

//
// =========================================================
// FILTERED OUT IMAGES
// =========================================================
//

type FilteredOut struct {
	ID *uint64

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

	LastSeenSync *uint64
}

func (i *FilteredOut) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("path", i.Path).
			Str("filename", i.Filename).
			Str("ext", i.Ext).
			Str("root", i.Root)
		logging.Uint64If(e, "id", i.ID)
		logging.Uint64If(e, "last sync", i.LastSeenSync)

	}
	if level == zerolog.TraceLevel {
		e.Str("file_hash", i.FileHash).
			Str("meta_hash", i.MetaHash).
			Time("created_at", i.CreatedAt).
			Time("updated_at", i.UpdatedAt)

	}
}
