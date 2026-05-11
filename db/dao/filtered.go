package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
)

const filteredFields = `
f.id, f.root, f.path, f.filename, f.ext,
f.file_size, f.mtime, f.file_hash, f.meta_hash,
f.exif_json,
f.created_at, f.updated_at, f.last_seen_sync
`
const getFilteredByPath = `
SELECT ` + filteredFields + ` FROM filtered f WHERE f.root=? AND f.path = ? AND f.filename = ? AND f.ext = ?`

const deleteFilteredNotSeen = `DELETE FROM filtered WHERE (last_seen_sync IS NULL OR last_seen_sync <> ?)
LIMIT ?`

const updateFilteredSyncIdByPath = `UPDATE filtered f SET f.last_seen_sync=? WHERE f.root=? AND f.path = ? AND f.filename = ? AND f.ext = ?`

const createFiltered = `
INSERT INTO filtered (
  root, path, filename, ext,
  file_size, mtime, file_hash, meta_hash,
  exif_json,
  last_seen_sync
) VALUES (?,?,?,?,?,?,?,?,?,?)`

const checkFiltered = `
SELECT 1 FROM filtered f WHERE
f.root=? AND f.path = ? AND f.filename = ? AND f.ext = ?
`

func parseFiltered(row *sql.Row) (dbo.FilteredOut, error) {
	var f dbo.FilteredOut
	err := row.Scan(
		&f.ID,
		&f.Root,
		&f.Path,
		&f.Filename,
		&f.Ext,
		&f.FileSize,
		&f.MTime,
		&f.FileHash,
		&f.MetaHash,
		&f.ExifJSON,
		&f.CreatedAt,
		&f.UpdatedAt,
		&f.LastSeenSync,
	)
	return f, err
}
func (q *Queries) GetFilteredByPath(ctx context.Context, root, path, filename, ext string) (dbo.FilteredOut, error) {
	row := q.db.QueryRowContext(ctx, getFilteredByPath, root, path, filename, ext)
	return parseFiltered(row)
}

func (q *Queries) DeleteFilteredNotSeen(ctx context.Context, syncID uint64, limit uint32) (uint64, error) {
	res, err := q.db.ExecContext(ctx, deleteFilteredNotSeen, syncID, limit)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint64(affected), nil
}
func (q *Queries) UpdateFilteredSyncIdByPath(ctx context.Context, root, path, filename, ext string, syncId uint64) error {
	_, err := q.db.ExecContext(ctx, updateFilteredSyncIdByPath, syncId, root, path, filename, ext)
	return err
}

func (q *Queries) CreateFiltered(ctx context.Context, f dbo.FilteredOut) error {
	_, err := q.db.ExecContext(
		ctx,
		createFiltered,
		f.Root,
		f.Path,
		f.Filename,
		f.Ext,
		f.FileSize,
		f.MTime,
		f.FileHash,
		f.MetaHash,
		f.ExifJSON,
		f.LastSeenSync,
	)
	return err
}

func (q *Queries) CheckFilteredByPath(ctx context.Context, root, path, filename, ext string) error {
	row := q.db.QueryRowContext(ctx, checkFiltered, root, path, filename, ext)
	var dummy int
	return row.Scan(&dummy)
}

func GetFilteredByPath(db *sql.DB, c context.Context, root, path, filename, ext string) (dbo.FilteredOut, error) {
	logScope, ctx := logging.Enter(c, "dao/filtered/get/byPath", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
	})
	q := NewQueries(db)
	f, err := q.GetFilteredByPath(ctx, root, path, filename, ext)
	return f, returnWrapNotFound(logScope, err, "filtered")
}

func DeleteFilteredNotSeen(db *sql.DB, c context.Context, syncID uint64, limit uint32) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/filtered/delete/notSeen", syncID, map[string]any{"sync_id": syncID})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	deleted, err := q.DeleteFilteredNotSeen(ctx, syncID, limit)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	return deleted, logging.ReturnParams(logScope, tx.Commit(), map[string]any{"deleted": deleted})
}
func DeleteFilteredNotSeenAll(db *sql.DB, c context.Context, syncID uint64, limit uint32) error {
	logScope, ctx := logging.Enter(c, "dao/filtered/delete/notSeen/all", syncID, map[string]any{"sync_id": syncID, "limit": limit})
	deleted, err := DeleteFilteredNotSeen(db, ctx, syncID, limit)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	batch := 1
	for deleted > 0 {
		logging.Debug(logScope, "loop", map[string]any{
			"batch": batch,
		})
		deleted, err = DeleteFilteredNotSeen(db, ctx, syncID, limit)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		batch++
	}
	logging.Exit(logScope, "ok", map[string]any{"batch": batch})
	return nil
}

func UpdateFilteredSyncIdByPath(db *sql.DB, c context.Context, root, path, filename, ext string, syncID uint64) error {
	logScope, ctx := logging.Enter(c, "dao/filtered/update/syncID/byPath", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
		"sync_id":  syncID,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	err = q.UpdateFilteredSyncIdByPath(ctx, root, path, filename, ext, syncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}
func CreateFiltered(db *sql.DB, c context.Context, f *dbo.FilteredOut) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/filtered/create", f.Root+"/"+f.Path+"/"+f.Filename+"."+f.Ext, map[string]any{
		"root":     f.Root,
		"path":     f.Path,
		"filename": f.Filename,
		"ext":      f.Ext,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateFiltered(ctx, *f); err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}

	id, err := q.GetLastId(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	f.ID = utils.PtrUint64(uint64(id))

	return uint64(id), logging.ReturnParams(logScope, tx.Commit(), map[string]any{"new_ID": id})
}
func CreateOrUpdateFiltered(db *sql.DB, c context.Context, f *dbo.FilteredOut) error {
	logScope, ctx := logging.Enter(c, "dao/filtered/createOrUpdate", f.Root+"/"+f.Path+"/"+f.Filename+"."+f.Ext, map[string]any{
		"root":     f.Root,
		"path":     f.Path,
		"filename": f.Filename,
		"ext":      f.Ext,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	found := false
	err = q.CheckFilteredByPath(ctx, f.Root, f.Path, f.Filename, f.Ext)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		found = false
	case err != nil:
		logging.ExitErr(logScope, err)
		return err
	default:
		found = true
	}
	if found {
		if err := q.UpdateFilteredSyncIdByPath(ctx, f.Root, f.Path, f.Filename, f.Ext, *f.LastSeenSync); err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
	} else {
		if err := q.CreateFiltered(ctx, *f); err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		id, err := q.GetLastId(ctx)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		f.ID = utils.PtrUint64(uint64(id))
	}
	return logging.Return(logScope, tx.Commit())
}
