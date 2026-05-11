package dbo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/rs/zerolog"
)

type SyncMode string
type TagSource string
type ImageFocusMode string
type DBACLLevel int
type ACLScope string

type ACLRole string

func (r ACLRole) Compare(role ACLRole) int {
	if r == role {
		return 0
	}
	switch r {
	case RoleGuest:
		return -1
	case RoleAdmin:
		return 1
	}
	if role == RoleGuest {
		return 1
	}
	return -1
}

func ParseRole(s string) (ACLRole, error) {
	switch strings.ToLower(s) {
	case "guest":
		return RoleGuest, nil
	case "user":
		return RoleUser, nil
	case "admin":
		return RoleAdmin, nil
	default:
		return "", fmt.Errorf("invalid role: %s", s)
	}
}

func IsValidRole(s ACLRole) bool {
	switch strings.ToLower(string(s)) {
	case string(RoleGuest), string(RoleUser), string(RoleAdmin):
		return true
	default:
		return false
	}
}

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

	DBACLLevelPublic        DBACLLevel = 0
	DBACLLevelAuthenticated DBACLLevel = 1
	DBACLLevelUser          DBACLLevel = 1
	DBACLLevelAdmin         DBACLLevel = 2

	ACLScopePublic        ACLScope = "public"
	ACLScopeAuthenticated ACLScope = "authenticated"
	ACLScopeUser          ACLScope = "user"
	ACLScopeAdmin         ACLScope = "admin"

	RoleGuest ACLRole = "guest"
	RoleUser  ACLRole = "user"
	RoleAdmin ACLRole = "admin"
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
// SYNC
// =========================================================
//

type SyncStatus string

const (
	SyncStatusRunning  SyncStatus = "running"
	SyncStatusFinished SyncStatus = "finished"
	SyncStatusFailed   SyncStatus = "failed"
)

type SyncRun struct {
	ID           *uint64
	IsActive     *bool
	StartedAt    time.Time
	FinishedAt   *time.Time
	Mode         SyncMode
	TotalSeen    uint32
	TotalDeleted uint32
	Status       SyncStatus
	Error        *string
	MetaHash     *string
}

func (s *SyncRun) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		logging.Uint64If(e, "id", s.ID)
		logging.BoolIf(e, "is_active", s.IsActive)

		e.Time("started_at", s.StartedAt).
			Str("mode", string(s.Mode)).
			Uint32("total_seen", s.TotalSeen).
			Uint32("total_deleted", s.TotalDeleted).
			Str("status", string(s.Status))

		logging.TimeIf(e, "finished_at", s.FinishedAt)
		logging.StrIf(e, "error", s.Error)
		logging.StrIf(e, "meta_sha", s.MetaHash)
	}
}

type SyncFileStatus string

const (
	SyncFileStatusDeleted     SyncFileStatus = "deleted"
	SyncFileStatusCreated     SyncFileStatus = "created"
	SyncFileStatusNotChanged  SyncFileStatus = "not_changed"
	SyncFileStatusUpdated     SyncFileStatus = "updated"
	SyncFileStatusFilteredOut SyncFileStatus = "filtered_out"
	SyncFileStatusError       SyncFileStatus = "error"
)

var (
	AllSyncFileStatus = []SyncFileStatus{
		SyncFileStatusDeleted,
		SyncFileStatusCreated,
		SyncFileStatusNotChanged,
		SyncFileStatusUpdated,
		SyncFileStatusFilteredOut,
		SyncFileStatusError,
	}

// SyncFileStatusForced      SyncFileStatus = "forced"
)

type SyncFile struct {
	ID     *uint64
	SyncID uint64

	Root     string
	Path     string
	Filename string
	Ext      string

	Status      SyncFileStatus
	DirtyReason *string

	RuleResultsJSON []byte

	CreatedAt time.Time
}

func (s *SyncFile) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		logging.Uint64If(e, "id", s.ID)

		e.Uint64("sync_id", s.SyncID).
			Str("root", s.Root).
			Str("path", s.Path).
			Str("filename", s.Filename).
			Str("ext", s.Ext).
			Str("status", string(s.Status))

		logging.StrIf(e, "dirty_reason", s.DirtyReason)

		e.Time("created_at", s.CreatedAt)

		if level <= zerolog.TraceLevel {
			if len(s.RuleResultsJSON) > 0 {
				e.RawJSON("results", s.RuleResultsJSON)
			}
		}
	}
}
func (s SyncFile) PathFull() string {
	return BuildFullPath(s.Root, s.Path, s.Filename, s.Ext)
}

//
// =========================================================
// MIXED
// =========================================================
//

type ImageWLastSyncWUser struct {
	Image
	LastSyncDate *time.Time
	User         *string
}

func ParseACLScope(s string) (DBACLLevel, error) {
	scope := ACLScope(s)

	switch scope {
	case ACLScopePublic:
		return DBACLLevelPublic, nil
	case ACLScopeAuthenticated:
		return DBACLLevelAuthenticated, nil
	case ACLScopeUser:
		return DBACLLevelUser, nil
	case ACLScopeAdmin:
		return DBACLLevelAdmin, nil
	default:
		return DBACLLevelAdmin, fmt.Errorf("invalid ACL scope: %s", s)
	}
}
func ValidateACLLevel(level DBACLLevel) bool {
	switch level {
	case DBACLLevelAdmin,
		DBACLLevelAuthenticated,
		DBACLLevelPublic:
		return true
	default:
		return false
	}
}

type ACLContext struct {
	ViewerUserID *uint64
	Role         ACLRole
}

func (a *ACLContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("role", string(a.Role))
		logging.Uint64If(e, "userID", a.ViewerUserID)
	}
}
func BuildFullPath(root, path, filename, ext string) string {
	ret := ""
	ret += root + "/"
	if path != "" {
		ret += path + "/"
	}
	ret += filename + "." + ext
	return ret
}
