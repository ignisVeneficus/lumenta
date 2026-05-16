package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
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

// parseFiltered scans a single filtered row into a FilteredOut value.
//
// Input:
//   - row: SQL row containing the filteredFields columns.
//
// Output:
//   - dbo.FilteredOut: scanned filtered record.
//   - error: scan error, if any.
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

// GetFilteredByPath reads a filtered record by filesystem identity.
//
// Input:
//   - ctx: request context.
//   - root, path, filename, ext: filesystem identity fields.
//
// Output:
//   - dbo.FilteredOut: matching filtered record.
//   - error: query or scan error, including sql.ErrNoRows when not found.
func (q *Queries) GetFilteredByPath(ctx context.Context, root, path, filename, ext string) (dbo.FilteredOut, error) {
	row := q.db.QueryRowContext(ctx, getFilteredByPath, root, path, filename, ext)
	return parseFiltered(row)
}

// DeleteFilteredNotSeen deletes filtered records not seen in the given sync run.
//
// Input:
//   - ctx: request context.
//   - syncID: sync run used as the current seen marker.
//   - limit: maximum number of rows to delete.
//
// Output:
//   - uint64: number of deleted rows.
//   - error: exec or row-count error, if any.
func (q *Queries) DeleteFilteredNotSeen(ctx context.Context, syncID dbo.SyncRunID, limit uint32) (uint64, error) {
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

// UpdateFilteredSyncIDByPath updates the last seen sync marker for a filtered record.
//
// Input:
//   - ctx: request context.
//   - root, path, filename, ext: filesystem identity fields.
//   - syncID: sync run to store as last_seen_sync.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateFilteredSyncIDByPath(ctx context.Context, root, path, filename, ext string, syncID dbo.SyncRunID) error {
	_, err := q.db.ExecContext(ctx, updateFilteredSyncIdByPath, syncID, root, path, filename, ext)
	return err
}

// CreateFiltered inserts a filtered record.
//
// Input:
//   - ctx: request context.
//   - f: filtered record to insert.
//
// Output:
//   - error: exec error, if any.
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

// CheckFilteredByPath checks whether a filtered record exists by filesystem identity.
//
// Input:
//   - ctx: request context.
//   - root, path, filename, ext: filesystem identity fields.
//
// Output:
//   - error: nil when a row exists; sql.ErrNoRows when not found; scan error otherwise.
func (q *Queries) CheckFilteredByPath(ctx context.Context, root, path, filename, ext string) error {
	row := q.db.QueryRowContext(ctx, checkFiltered, root, path, filename, ext)
	var dummy int
	return row.Scan(&dummy)
}

// GetFilteredByPath reads a filtered record by filesystem identity with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path, filename, ext: filesystem identity fields.
//
// Output:
//   - dbo.FilteredOut: matching filtered record.
//   - error: wrapped not-found, query, or scan error.
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

// DeleteFilteredNotSeen deletes one batch of filtered records not seen in a sync run.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - syncID: sync run used as the current seen marker.
//   - limit: maximum number of rows to delete.
//
// Output:
//   - uint64: number of deleted rows.
//   - error: transaction, delete, or commit error.
func DeleteFilteredNotSeen(db *sql.DB, c context.Context, syncID dbo.SyncRunID, limit uint32) (uint64, error) {
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

// DeleteFilteredNotSeenAll deletes filtered records not seen in a sync run until none remain.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - syncID: sync run used as the current seen marker.
//   - limit: batch size for each delete pass.
//
// Output:
//   - error: delete error, if any.
func DeleteFilteredNotSeenAll(db *sql.DB, c context.Context, syncID dbo.SyncRunID, limit uint32) error {
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

// UpdateFilteredSyncIDByPath updates the last seen sync marker by filesystem identity with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path, filename, ext: filesystem identity fields.
//   - syncID: sync run to store as last_seen_sync.
//
// Output:
//   - error: transaction, update, or commit error.
func UpdateFilteredSyncIDByPath(db *sql.DB, c context.Context, root, path, filename, ext string, syncID dbo.SyncRunID) error {
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

	err = q.UpdateFilteredSyncIDByPath(ctx, root, path, filename, ext, syncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

// CreateFiltered inserts a filtered record and writes the new ID back to f.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - f: filtered record to insert; receives the new ID.
//
// Output:
//   - dbo.FilteredID: created filtered record ID.
//   - error: transaction, insert, ID lookup, or commit error.
func CreateFiltered(db *sql.DB, c context.Context, f *dbo.FilteredOut) (dbo.FilteredID, error) {
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
	newID := dbo.FilteredID(id)
	f.ID = &newID

	return newID, logging.ReturnParams(logScope, tx.Commit(), map[string]any{"new_ID": id})
}

// CreateOrUpdateFiltered creates a filtered record or updates its last seen sync marker.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - f: filtered record to create or update; receives a new ID on insert.
//
// Output:
//   - error: transaction, existence check, create, update, ID lookup, or commit error.
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
		if err := q.UpdateFilteredSyncIDByPath(ctx, f.Root, f.Path, f.Filename, f.Ext, *f.LastSeenSync); err != nil {
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
		newID := dbo.FilteredID(id)
		f.ID = &newID

	}
	return logging.Return(logScope, tx.Commit())
}
