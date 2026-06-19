package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/rs/zerolog/log"
)

const syncFileFields = `
f.id, f.sync_id,
f.root, f.path, f.filename, f.ext,
f.status, f.dirty_reason,
f.ruleresults_json,
f.created_at
`

const createSyncFile = `
INSERT INTO sync_files (
  sync_id,
  root, path, filename, ext,
  status, dirty_reason,
  ruleresults_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
const getSyncFileById = `SELECT ` + syncFileFields + ` FROM sync_files f WHERE f.id = ?`

const querySyncFileBySyncIDPagedStart = `
SELECT ` + syncFileFields + `
FROM sync_files f
WHERE f.sync_id = ?
`

const countSyncFileBySyncIDStart = `
SELECT count(*)
FROM sync_files f
WHERE f.sync_id = ?
`

const querySyncFileByPathPaged = `
SELECT ` + syncFileFields + `
FROM sync_files f
WHERE f.root = ? and f.path=? and f.filename = ? and f.ext = ?
ORDER BY f.created_at DESC
LIMIT ?, ?
`
const countSyncFileByPath = `
SELECT count(*)
FROM sync_files f
WHERE f.root = ? and f.path=? and f.filename = ? and f.ext = ?
`

const getSyncFileByPathByStatusLast = `
SELECT ` + syncFileFields + `
FROM sync_files f
WHERE f.root = ? and f.path=? and f.filename = ? and f.ext = ? and f.status = ?
ORDER BY f.created_at DESC
LIMIT 1
`
const querySyncFileBySearchByStatusPagedStart = `
SELECT ` + syncFileFields + `
FROM sync_files f
WHERE 1 = 1
`
const querySyncFileBySearchByStatusSearch = `
AND ( f.path LIKE CONCAT('%', ?, '%')
  OR f.filename LIKE CONCAT('%', ?, '%')
)
`
const querySyncFileByStatusMiddle = `
 AND f.status IN (%s)
`

const querySyncFilePagedEnd = `
ORDER BY f.root, f.path, f.filename, f.ext, f.created_at DESC
LIMIT ?,?
`

const countSyncFileBySearchByStatusStart = `
SELECT count(*)
FROM sync_files f
WHERE 1 = 1
`

const countSyncFilesStatusBySyncID = `
SELECT f.status, COUNT(*) from sync_files f 
WHERE f.sync_id = ?
GROUP BY f.status
`

const purgeSyncFileByType = `
DELETE FROM sync_run_files
WHERE id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY root, path, filename, ext
                   ORDER BY created_at DESC
               ) AS rn
        FROM sync_run_files
        WHERE status = ?
          AND run_id NOT IN (?,?,?,?,?)   -- last 5 run id
    ) t
    WHERE t.rn > 1
);`
const purgeSyncFileOtherStatus = `
DELETE FROM sync_run_files
WHERE status NOT IN ('created','updated','deleted','skipped')
  AND run_id NOT IN (?,?,?,?,?);
`

func parseSyncFile(row *sql.Row) (dbo.SyncFile, error) {
	var f dbo.SyncFile

	err := row.Scan(
		&f.ID,
		&f.SyncID,

		&f.Root,
		&f.Path,
		&f.Filename,
		&f.Ext,

		&f.Status,
		&f.DirtyReason,

		&f.RuleResultsJSON,

		&f.CreatedAt,
	)

	return f, err
}
func parseSyncFiles(rows *sql.Rows) ([]dbo.SyncFile, error) {
	out := make([]dbo.SyncFile, 0)
	for rows.Next() {
		var f dbo.SyncFile
		err := rows.Scan(
			&f.ID,
			&f.SyncID,

			&f.Root,
			&f.Path,
			&f.Filename,
			&f.Ext,

			&f.Status,
			&f.DirtyReason,

			&f.RuleResultsJSON,

			&f.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (q *Queries) CreateSyncFile(ctx context.Context, f dbo.SyncFile) error {
	_, err := q.db.ExecContext(ctx, createSyncFile,
		f.SyncID,
		f.Root, f.Path, f.Filename, f.Ext,
		f.Status, f.DirtyReason,
		f.RuleResultsJSON,
	)
	return err
}
func (q *Queries) GetSyncFileById(ctx context.Context, syncFileID dbo.SyncFileID) (dbo.SyncFile, error) {
	row := q.db.QueryRowContext(ctx, getSyncFileById, syncFileID)
	return parseSyncFile(row)
}

func (q *Queries) QuerySyncFileBySyncIDPaged(ctx context.Context, syncRunID dbo.SyncRunID, status []string, from, qty uint64) ([]dbo.SyncFile, error) {
	query := querySyncFileBySyncIDPagedStart
	args := []any{syncRunID}
	if len(status) > 0 {
		query += fmt.Sprintf(querySyncFileByStatusMiddle, Placeholder(len(status)))
		for _, s := range status {
			args = append(args, s)
		}
	}
	query += querySyncFilePagedEnd
	args = append(args, from, qty)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseSyncFiles(rows)
}

func (q *Queries) CountSyncFileBySyncID(ctx context.Context, syncRunID dbo.SyncRunID, status []string) (uint64, error) {
	query := countSyncFileBySyncIDStart
	args := []any{syncRunID}
	if len(status) > 0 {
		query += fmt.Sprintf(querySyncFileByStatusMiddle, Placeholder(len(status)))
		for _, s := range status {
			args = append(args, s)
		}
	}
	row := q.db.QueryRowContext(ctx, query, args...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) QuerySyncFileByFilePathPaged(ctx context.Context, root, path, filename, ext string, from, qty uint64) ([]dbo.SyncFile, error) {
	rows, err := q.db.QueryContext(ctx, querySyncFileByPathPaged, root, path, filename, ext, from, qty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseSyncFiles(rows)
}

func (q *Queries) CountSyncFileByPath(ctx context.Context, root, path, filename, ext string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countSyncFileByPath, root, path, filename, ext)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetSyncFileByPathByStatusLast(ctx context.Context, root, path, filename, ext string, status dbo.SyncFileStatus) (dbo.SyncFile, error) {
	row := q.db.QueryRowContext(ctx, getSyncFileByPathByStatusLast, root, path, filename, ext, status)
	return parseSyncFile(row)
}

func (q *Queries) QuerySyncFileBySearchByStatusPaged(ctx context.Context, search string, status []string, from, qty uint64) ([]dbo.SyncFile, error) {
	query := querySyncFileBySearchByStatusPagedStart
	args := []any{}
	if search != "" {
		query += querySyncFileBySearchByStatusSearch
		args = append(args, search, search)
	}
	if len(status) > 0 {
		query += fmt.Sprintf(querySyncFileByStatusMiddle, Placeholder(len(status)))
		for _, s := range status {
			args = append(args, s)
		}
	}
	query += querySyncFilePagedEnd
	args = append(args, from, qty)
	log.Logger.Info().Str("q", query).Any("args", args).Msg("query debug")

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseSyncFiles(rows)
}

func (q *Queries) CountSyncFileBySearchByStatus(ctx context.Context, search string, status []string) (uint64, error) {
	query := countSyncFileBySearchByStatusStart
	args := []any{}
	if search != "" {
		query += querySyncFileBySearchByStatusSearch
		args = append(args, search, search)
	}
	if len(status) > 0 {
		query += fmt.Sprintf(querySyncFileByStatusMiddle, Placeholder(len(status)))
		for _, s := range status {
			args = append(args, s)
		}
	}
	row := q.db.QueryRowContext(ctx, query, args...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) CountSyncFileStatusBySyncID(ctx context.Context, syncID dbo.SyncRunID) (map[dbo.SyncFileStatus]uint64, error) {
	rows, err := q.db.QueryContext(ctx, countSyncFilesStatusBySyncID, syncID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[dbo.SyncFileStatus]uint64, 0)
	for rows.Next() {
		var key dbo.SyncFileStatus
		var qty uint64
		if err := rows.Scan(&key, &qty); err != nil {
			return nil, err
		}
		out[key] = qty
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

//
// =========================================================
// Public API functions
// =========================================================
//

func CreateSyncFile(db *sql.DB, c context.Context, f dbo.SyncFile) error {
	logScope, ctx := logging.Enter(c, "dao/sync_file/create", f.PathFull(), map[string]any{
		"sync_id": f.SyncID,
		"root":    f.Root,
		"path":    f.Path,
		"file":    f.Filename,
		"ext":     f.Ext,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logScope.ExitErr(err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	if err := q.CreateSyncFile(ctx, f); err != nil {
		err = NormalizeSQLError(err)
		logScope.ExitErr(err)
		return err
	}

	return logScope.Return(tx.Commit())
}
func GetSyncFileById(db *sql.DB, c context.Context, syncFileID dbo.SyncFileID) (dbo.SyncFile, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/get/byId", syncFileID, map[string]any{
		"id": syncFileID,
	})
	q := NewQueries(db)
	i, err := q.GetSyncFileById(ctx, syncFileID)
	return i, returnWrapNotFound(logScope, err, "sync_file")
}

func QuerySyncFileBySyncIDPaged(db *sql.DB, c context.Context, syncRunID dbo.SyncRunID, stats []string, from, qty uint64) ([]dbo.SyncFile, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/query/bySyncId/paged", syncRunID, map[string]any{
		"syncId": syncRunID,
		"from":   from,
		"qty":    qty,
	})

	q := NewQueries(db)
	files, err := q.QuerySyncFileBySyncIDPaged(ctx, syncRunID, stats, from, qty)
	if err != nil {
		logScope.ExitErr(err)
		return nil, err
	}

	logScope.Exit("ok", map[string]any{
		"found": len(files),
	})

	return files, nil
}

func CountSyncFileBySyncID(db *sql.DB, c context.Context, syncRunID dbo.SyncRunID, status []string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/count/bySyncId", syncRunID, map[string]any{
		"syncId": syncRunID,
	})
	q := NewQueries(db)
	qty, err := q.CountSyncFileBySyncID(ctx, syncRunID, status)
	if err != nil {
		logScope.ExitErr(err)
		return 0, err
	}
	logScope.Exit("ok", map[string]any{"return": qty})
	return qty, nil
}

func QuerySyncFileByFilePathPaged(db *sql.DB, c context.Context, root, path, filename, ext string, from, qty uint64) ([]dbo.SyncFile, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/query/byPath/paged", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
		"from":     from,
		"qty":      qty,
	})

	q := NewQueries(db)
	files, err := q.QuerySyncFileByFilePathPaged(ctx, root, path, filename, ext, from, qty)
	if err != nil {
		logScope.ExitErr(err)
		return nil, err
	}

	logScope.Exit("ok", map[string]any{
		"found": len(files),
	})

	return files, nil
}

func CountSyncFileByPath(db *sql.DB, c context.Context, root, path, filename, ext string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/count/byPath", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
	})
	q := NewQueries(db)
	qty, err := q.CountSyncFileByPath(ctx, root, path, filename, ext)
	if err != nil {
		logScope.ExitErr(err)
		return 0, err
	}
	logScope.Exit("ok", map[string]any{"return": qty})
	return qty, nil
}
func GetSyncFileByPathByStatusLast(db *sql.DB, c context.Context, root, path, filename, ext string, status dbo.SyncFileStatus) (dbo.SyncFile, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/get/byId", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
		"status":   status,
	})
	q := NewQueries(db)
	i, err := q.GetSyncFileByPathByStatusLast(ctx, root, path, filename, ext, status)
	return i, returnWrapNotFound(logScope, err, "sync_file")
}

func QuerySyncFileBySearchByStatusPaged(db *sql.DB, c context.Context, search string, status []string, from, qty uint64) ([]dbo.SyncFile, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/query/bySearch/byStatus/paged", nil, map[string]any{
		"search": search,
		"status": status,
		"from":   from,
		"qty":    qty,
	})

	q := NewQueries(db)
	files, err := q.QuerySyncFileBySearchByStatusPaged(ctx, search, status, from, qty)
	if err != nil {
		logScope.ExitErr(err)
		return nil, err
	}

	logScope.Exit("ok", map[string]any{
		"found": len(files),
	})

	return files, nil
}

func CountSyncFileBySearchByStatus(db *sql.DB, c context.Context, search string, status []string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/count/byPath", nil, map[string]any{
		"search": search,
		"status": status,
	})
	q := NewQueries(db)
	qty, err := q.CountSyncFileBySearchByStatus(ctx, search, status)
	if err != nil {
		logScope.ExitErr(err)
		return 0, err
	}
	logScope.Exit("ok", map[string]any{"return": qty})
	return qty, nil
}

func CountSyncFileStatusBySyncID(db *sql.DB, c context.Context, syncID dbo.SyncRunID) (map[dbo.SyncFileStatus]uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/sync_file/count/status/bySyncID", syncID, map[string]any{
		"sync_id": syncID,
	})
	q := NewQueries(db)
	count, err := q.CountSyncFileStatusBySyncID(ctx, syncID)
	if err != nil {
		logScope.ExitErr(err)
		return nil, err
	}
	logScope.Exit("ok", nil)
	return count, nil
}
