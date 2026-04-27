package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

//
// =========================================================
// IMAGES
// =========================================================
//

type Image struct {
	ID *uint64

	Root     string
	Path     string
	Filename string
	Ext      string

	FileSize uint64
	MTime    time.Time
	FileHash string
	MetaHash string

	Title   *string
	Caption *string

	TakenAt     *time.Time
	Camera      *string
	Lens        *string
	FocalLength *float32
	Aperture    *float32
	Exposure    *float64
	ISO         *uint16

	Latitude  *float64
	Longitude *float64

	Rotation *int16
	Rating   *uint16

	Width  uint32
	Height uint32

	Panorama int8

	FocusX    *float32
	FocusY    *float32
	FocusMode ImageFocusMode

	ExifJSON json.RawMessage

	ACLLevel  DBACLLevel
	ACLUserID uint64

	CreatedAt time.Time
	UpdatedAt time.Time

	LastSeenSync *uint64

	Tags TagsTree
}

func (i Image) GetTitle() string {
	if i.Title != nil {
		return *i.Title
	}
	return i.Filename
}
func (i Image) PathFull() string {
	return BuildFullPath(i.Root, i.Path, i.Filename, i.Ext)
}

func (i *Image) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("path", i.Path).
			Str("filename", i.Filename).
			Str("ext", i.Ext).
			Uint64("acl_level", uint64(i.ACLLevel)).
			Uint64("acl_user_id", i.ACLUserID).
			Str("root", i.Root)
		logging.Uint64If(e, "id", i.ID)
		logging.StrIf(e, "title", i.Title)
		logging.Uint64If(e, "last sync", i.LastSeenSync)

	}
	if level == zerolog.TraceLevel {
		e.Str("file_hash", i.FileHash).
			Str("meta_hash", i.MetaHash).
			Time("created_at", i.CreatedAt).
			Time("updated_at", i.UpdatedAt).
			Uint32("width", i.Width).
			Uint32("height", i.Height).
			Int8("panorama", i.Panorama)

		logging.Uint16If(e, "rating", i.Rating)
		logging.TimeIf(e, "taken_at", i.TakenAt)
		logging.StrIf(e, "camera", i.Camera)
		logging.StrIf(e, "lens", i.Lens)
		logging.StrIf(e, "subject", i.Caption)
	}
}

func (i *Image) AddTags(tags []*Tag) {
	i.Tags = BuildTagsTree(tags)
}

type ImageTitle struct {
	ID    uint64
	Title string
}

type ImageCoord struct {
	ID        uint64
	Title     string
	Latitude  *float64
	Longitude *float64
}

type ImageACLCount map[DBACLLevel]uint64
