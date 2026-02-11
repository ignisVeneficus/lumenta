package dbo

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

type SyncMode string
type TagSource string
type ImageFocusMode string
type ACLScope string

const (
	SyncModeFull        SyncMode = "full"
	SyncModeIncremental SyncMode = "incremental"
	SyncModePartial     SyncMode = "partial"

	TagSourceDigikam TagSource = "digikam"

	ImageFocusModeAuto   ImageFocusMode = "auto"
	ImageFocusModeManual ImageFocusMode = "manual"
	ImageFocusModeCenter ImageFocusMode = "center"
	ImageFocusModeTop    ImageFocusMode = "top"
	ImageFocusModeBottom ImageFocusMode = "bottom"
	ImageFocusModeLeft   ImageFocusMode = "left"
	ImageFocusModeRight  ImageFocusMode = "right"

	ACLScopePublic  ACLScope = "public"
	ACLScopeAnyUser ACLScope = "any_user"
	ACLScopeUser    ACLScope = "user"
	ACLScopeGroup   ACLScope = "group"
	ACLScopeAdmin   ACLScope = "admin"
)

type User struct {
	ID           *uint64
	Username     string
	Email        *string
	Role         string
	HashPassword string
	Disabled     bool
	CreatedAt    time.Time
}

func (u *User) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("username", u.Username).
			Str("role", u.Role).
			Bool("disabled", u.Disabled)
		logging.Uint64If(e, "id", u.ID)
	}
	if level == zerolog.TraceLevel {
		logging.StrIf(e, "email", u.Email)
		e.Time("created_at", u.CreatedAt)
	}
}

type Album struct {
	ID          *uint64
	ParentID    *uint64
	Name        string
	Description *string

	Rank *uint64

	AncestorIDs  []uint64
	RuleJSON     json.RawMessage
	CoverImageID *uint64

	ChildAlbumCount uint32
	ImageCount      uint32

	ACLScope   ACLScope
	ACLUserID  *uint64
	ACLGroupID *uint64

	UpdatedAt time.Time

	Images []Image
}

func (a *Album) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", a.Name).
			Str("acl_scope", string(a.ACLScope)).
			Uint32("child_album_count", a.ChildAlbumCount).
			Uint32("image_count", a.ImageCount)

		logging.Uint64If(e, "parent_id", a.ParentID)
		logging.Uint64If(e, "cover_image_id", a.CoverImageID)
		logging.Uint64If(e, "id", a.ID)

	}
	if level == zerolog.TraceLevel {
		e.RawJSON("rule", a.RuleJSON).
			Time("updated_at", a.UpdatedAt)
		logging.Uint64If(e, "acl_user_id", a.ACLUserID)
		logging.Uint64If(e, "acl_group_id", a.ACLGroupID)
		logging.StrIf(e, "description", a.Description)
	}
}

func (a *Album) IsDescendantOf(parent *Album) bool {
	if len(a.AncestorIDs) < len(parent.AncestorIDs) {
		return false
	}
	for i := range parent.AncestorIDs {
		if a.AncestorIDs[i] != parent.AncestorIDs[i] {
			return false
		}
	}
	return true
}
func (a *Album) AncestorPrefixJSON() string {
	// pl: [1,5]
	if len(a.AncestorIDs) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(a.AncestorIDs)
	// levágjuk a záró ']'-t → "[1,5"
	return string(b[:len(b)-1])
}

func (a *Album) IsRoot() bool {
	return a.ParentID == nil
}

func ReplaceAncestorPrefix(oldAncestors, newAncestors, current []uint64) []uint64 {
	if len(current) < len(oldAncestors) {
		return current
	}
	return append(
		append([]uint64{}, newAncestors...),
		current[len(oldAncestors):]...,
	)
}
func BuildAncestorIDs(parent *Album, selfID uint64) []uint64 {
	if parent == nil {
		return []uint64{selfID}
	}

	out := make([]uint64, 0, len(parent.AncestorIDs)+1)
	out = append(out, parent.AncestorIDs...)
	out = append(out, selfID)
	return out
}

//
// =========================================================
// GROUPS
// =========================================================
//

type Group struct {
	ID    uint64
	Name  string
	Users []User
}

func (g *Group) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Uint64("id", g.ID).
			Str("name", g.Name)
	}
}

//
// =========================================================
// IMAGES
// =========================================================
//

type Image struct {
	ID *uint64

	Path     string
	Filename string
	Ext      string

	FileSize uint64
	MTime    time.Time
	FileHash string
	MetaHash string

	Title   *string
	Subject *string

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

	ACLScope   ACLScope
	ACLUserID  *uint64
	ACLGroupID *uint64

	CreatedAt time.Time
	UpdatedAt time.Time

	LastSeenSync *uint64

	Tags []Tag
}

func (i *Image) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("path", i.Path).
			Str("filename", i.Filename).
			Str("ext", i.Ext).
			Str("acl_scope", string(i.ACLScope))
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
		logging.Uint64If(e, "acl_user_id", i.ACLUserID)
		logging.Uint64If(e, "acl_group_id", i.ACLGroupID)
		logging.StrIf(e, "subject", i.Subject)
	}
}

//
// =========================================================
// TAGS
// =========================================================
//

type Tag struct {
	ID       *uint64
	Name     string
	ParentID *uint64
	Source   TagSource
	Children []Tag
}

func (t *Tag) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("name", t.Name).
			Str("source", string(t.Source))
		logging.Uint64If(e, "id", t.ID)
		logging.Uint64If(e, "parent_id", t.ParentID)
	}
}

type ImageWLastSyncWUser struct {
	Image
	LastSyncDate *time.Time
	User         *string
}
