package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/rs/zerolog"
)

// =========================================================
// IMAGES
// =========================================================
type ImageID uint64

type ValueSource string
type ImageFocusMode string

var (
	ValueSourceFilesystem ValueSource = "filesystem"
	ValueSourceUser       ValueSource = "user"

	ImageFocusModeAuto   ImageFocusMode = "auto"
	ImageFocusModeManual ImageFocusMode = "manual"
	ImageFocusModeCenter ImageFocusMode = "center"
	ImageFocusModeTop    ImageFocusMode = "top"
	ImageFocusModeBottom ImageFocusMode = "bottom"
	ImageFocusModeLeft   ImageFocusMode = "left"
	ImageFocusModeRight  ImageFocusMode = "right"
)

type Image struct {
	ID *ImageID

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

	FocusX      *float32
	FocusY      *float32
	FocusMode   ImageFocusMode
	FocusSource ValueSource

	ExifJSON json.RawMessage

	ACLLevel  DBACLLevel
	ACLUserID UserID
	ACLSource ValueSource

	CreatedAt time.Time
	UpdatedAt time.Time

	LastSeenSync *SyncRunID

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
			Uint64("acl_user_id", uint64(i.ACLUserID)).
			Str("root", i.Root).
			Str("acl_source", string(i.ACLSource))

		logging.Uint64If(e, "id", (*uint64)(i.ID))
		logging.StrIf(e, "title", i.Title)
		logging.Uint64If(e, "last sync", (*uint64)(i.LastSeenSync))

	}
	if level == zerolog.TraceLevel {
		e.Str("file_hash", i.FileHash).
			Str("meta_hash", i.MetaHash).
			Time("created_at", i.CreatedAt).
			Time("updated_at", i.UpdatedAt).
			Uint32("width", i.Width).
			Uint32("height", i.Height).
			Int8("panorama", i.Panorama).
			Str("focus_source", string(i.FocusSource)).
			Str("focus_mode", string(i.FocusMode))

		logging.Uint16If(e, "rating", i.Rating)
		logging.TimeIf(e, "taken_at", i.TakenAt)
		logging.StrIf(e, "camera", i.Camera)
		logging.StrIf(e, "lens", i.Lens)
		logging.StrIf(e, "subject", i.Caption)
		logging.Float32If(e, "focus_x", i.FocusX)
		logging.Float32If(e, "focus_y", i.FocusY)
	}
}

func ValidateImageFocusMode(focus ImageFocusMode) bool {
	switch focus {
	case ImageFocusModeAuto, ImageFocusModeBottom, ImageFocusModeCenter,
		ImageFocusModeLeft, ImageFocusModeManual, ImageFocusModeRight, ImageFocusModeTop:
		return true
	default:
		return false
	}

}
func ValidateImageFocus(focus *float32) bool {
	if focus == nil {
		return true
	}
	return (*focus >= 0 && *focus <= 1)
}

type ImageTitle struct {
	ID    ImageID
	Title string
}

type ImageCoord struct {
	ID        ImageID
	Title     string
	Latitude  *float64
	Longitude *float64
}

type ImageACLCount map[DBACLLevel]uint64
