package dao

import (
	"context"
	"database/sql"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
)

const syncRunFields = `s.id, s.is_active, s.started_at, s.finished_at, s.mode, s.total_seen, s.total_deleted, s.status, s.error, s.meta_hash `

const createSyncRun = `INSERT INTO sync_runs (started_at, mode, meta_hash) VALUES (NOW(), ?, ?)`

const closeSyncRunSuccess = `UPDATE sync_runs SET
  finished_at = NOW(),
  status = 'finished',
  total_seen = ?,
  total_deleted = ?,
  is_active = null
WHERE id = ?`

const closeSyncRunError = `UPDATE sync_runs
SET
  finished_at = NOW(),
  status = 'failed',
  error = ?,
  is_active = null
WHERE id = ?`

const getSyncRunLastHash = "SELECT meta_hash FROM sync_runs WHERE status='finished' ORDER BY started_at desc LIMIT 1"

const getSyncRunByID = `SELECT ` + syncRunFields + ` FROM sync_runs s WHERE s.id = ?`

const querySyncRunPaged = `SELECT ` + syncRunFields + ` FROM sync_runs s ORDER BY s.started_at DESC LIMIT ?,? `
const countSyncRun = `SELECT count(*) FROM sync_runs s`

/*
UPDATE sync_runs
SET
  status = 'failed',
  finished_at = NOW(),
  is_active = 0,
  error = 'stale sync reset'
WHERE
  is_active = 1
  AND started_at < NOW() - INTERVAL 2 HOUR;
*/

func parseSyncRunRow(row *sql.Row) (dbo.SyncRun, error) {
	var s dbo.SyncRun
	err := row.Scan(&s.ID, &s.IsActive, &s.StartedAt, &s.FinishedAt, &s.Mode, &s.TotalSeen, &s.TotalDeleted, &s.Status, &s.Error, &s.MetaHash)
	return s, err
}

func parseSyncRunRows(rows *sql.Rows) ([]dbo.SyncRun, error) {
	out := make([]dbo.SyncRun, 0)
	for rows.Next() {
		var s dbo.SyncRun
		err := rows.Scan(&s.ID, &s.IsActive, &s.StartedAt, &s.FinishedAt, &s.Mode, &s.TotalSeen, &s.TotalDeleted, &s.Status, &s.Error, &s.MetaHash)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (q *Queries) GetSyncRunByID(ctx context.Context, id uint64) (dbo.SyncRun, error) {
	row := q.db.QueryRowContext(ctx, getSyncRunByID, id)
	return parseSyncRunRow(row)
}

func (q *Queries) CreateRunSync(ctx context.Context, mode dbo.SyncMode, metaHash string) error {
	_, err := q.db.ExecContext(ctx, createSyncRun, mode, metaHash)
	return err
}

func (q *Queries) CloseSyncRunSuccess(ctx context.Context, syncID uint64, totalSeen uint64, totalDeleted uint64) error {
	_, err := q.db.ExecContext(ctx, closeSyncRunSuccess, totalSeen, totalDeleted, syncID)
	return err
}

func (q *Queries) CloseSyncRunError(ctx context.Context, syncID uint64, errorMsg string) error {
	_, err := q.db.ExecContext(ctx, closeSyncRunError, errorMsg, syncID)
	return err
}

func (q *Queries) GetSyncRunLastHash(ctx context.Context) (string, error) {
	row := q.db.QueryRowContext(ctx, getSyncRunLastHash)
	var hash string
	err := row.Scan(&hash)
	return hash, err
}

func (q *Queries) QuerySyncRunPaged(ctx context.Context, from, qty uint64) ([]dbo.SyncRun, error) {
	rows, err := q.db.QueryContext(ctx, querySyncRunPaged, from, qty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return parseSyncRunRows(rows)
}
func (q *Queries) CountSyncRun(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countSyncRun)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

//
// =========================================================
// Public API functions
// =========================================================
//

func GetSyncRunByID(db *sql.DB, ctx context.Context, id uint64) (dbo.SyncRun, error) {
	logg := logging.Enter(ctx, "dao.sync_run.getById", map[string]any{"id": id})
	q := NewQueries(db)

	s, err := q.GetSyncRunByID(ctx, id)
	return s, returnWrapNotFound(logg, err, "sync_run")
}

func CreateSyncRun(db *sql.DB, ctx context.Context, mode dbo.SyncMode, metaHash string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.sync_run.create", map[string]any{"mode": mode, "meta_hash": metaHash})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err = q.CreateRunSync(ctx, mode, metaHash); err != nil {
		err = NormalizeSQLError(err)
		logging.ExitErr(logg, err)
		return 0, err
	}
	id, err := q.GetLastId(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	return id, logging.Return(logg, tx.Commit())
}
func CloseSyncRunSuccess(db *sql.DB, ctx context.Context, syncID uint64, totalSeen uint64, totalDeleted uint64) error {
	logg := logging.Enter(ctx, "dao.sync_run.close.success", map[string]any{"sync_id": syncID})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncRunSuccess(ctx, syncID, totalSeen, totalDeleted); err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	return logging.Return(logg, tx.Commit())
}
func CloseSyncRunError(db *sql.DB, ctx context.Context, syncID uint64, errorMsg string) error {
	logg := logging.Enter(ctx, "dao.sync_run.close.error", map[string]any{"sync_id": syncID, "error": errorMsg})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncRunError(ctx, syncID, errorMsg); err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	return logging.Return(logg, tx.Commit())
}
func GetSyncRunLastHash(db *sql.DB, ctx context.Context) (string, error) {
	logg := logging.Enter(ctx, "dao.sync_run.meta_hash", nil)
	q := NewQueries(db)
	hash, err := q.GetSyncRunLastHash(ctx)
	return hash, returnWrapNotFound(logg, err, "sync_run")
}
func QuerySyncRunPaged(db *sql.DB, ctx context.Context, from, qty uint64) ([]dbo.SyncRun, error) {
	logg := logging.Enter(ctx, "dao.sync_run.query.PathPaged", map[string]any{
		"from": from,
		"qty":  qty,
	})

	q := NewQueries(db)

	runs, err := q.QuerySyncRunPaged(ctx, from, qty)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}

	logging.Exit(logg, "ok", map[string]any{
		"found": len(runs),
	})

	return runs, nil
}

func CountSyncRun(db *sql.DB, ctx context.Context) (uint64, error) {
	logg := logging.Enter(ctx, "dao.sync_run.count", nil)
	q := NewQueries(db)
	qty, err := q.CountSyncRun(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil

}
