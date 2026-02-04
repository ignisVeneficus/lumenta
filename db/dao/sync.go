package dao

import (
	"context"
	"database/sql"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
)

const createSync = `INSERT INTO sync_runs (started_at, mode, meta_hash) VALUES (NOW(), ?, ?)`

const closeSyncSuccess = `UPDATE sync_runs SET
  finished_at = NOW(),
  status = 'finished',
  total_seen = ?,
  total_deleted = ?,
  is_active = null
WHERE id = ?`

const closeSyncError = `UPDATE sync_runs
SET
  finished_at = NOW(),
  status = 'failed',
  error = ?,
  is_active = null
WHERE id = ?`

const getSyncRunLastHash = "SELECT meta_hash FROM sync_runs WHERE status='finished' ORDER BY started_at desc LIMIT 1"

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

func (q *Queries) CreateSync(ctx context.Context, mode dbo.SyncMode, metaHash string) error {
	_, err := q.db.ExecContext(ctx, createSync, mode, metaHash)
	return err
}

func (q *Queries) CloseSyncSuccess(ctx context.Context, syncID uint64, totalSeen uint64, totalDeleted uint64) error {
	_, err := q.db.ExecContext(ctx, closeSyncSuccess, totalSeen, totalDeleted, syncID)
	return err
}

func (q *Queries) CloseSyncError(ctx context.Context, syncID uint64, errorMsg string) error {
	_, err := q.db.ExecContext(ctx, closeSyncError, errorMsg, syncID)
	return err
}

func (q *Queries) GetSyncRunLastHash(ctx context.Context) (string, error) {
	row := q.db.QueryRowContext(ctx, getSyncRunLastHash)
	var hash string
	err := row.Scan(&hash)
	return hash, err
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
	if err = q.CreateSync(ctx, mode, metaHash); err != nil {
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
	logg := logging.Enter(ctx, "dao.sync_run.success", map[string]any{"sync_id": syncID})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncSuccess(ctx, syncID, totalSeen, totalDeleted); err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	return logging.Return(logg, tx.Commit())
}
func CloseSyncRunError(db *sql.DB, ctx context.Context, syncID uint64, errorMsg string) error {
	logg := logging.Enter(ctx, "dao.sync_run.error", map[string]any{"sync_id": syncID, "error": errorMsg})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncError(ctx, syncID, errorMsg); err != nil {
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
