package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/rs/zerolog/log"
)

const createSync = `INSERT INTO sync_runs (started_at, mode) VALUES (NOW(), ?)`

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

func (q *Queries) CreateSync(ctx context.Context, mode dbo.SyncMode) error {
	_, err := q.db.ExecContext(ctx, createSync, mode)
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

func CreateSyncRun(db *sql.DB, ctx context.Context, mode dbo.SyncMode) (uint64, error) {
	log.Logger.Debug().Str("mode", string(mode)).Msg("Create sync Run")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err = q.CreateSync(ctx, mode); err != nil {
		return 0, NormalizeSQLError(err)
	}
	id, err := q.GetLastId(ctx)
	if err != nil {
		return 0, fmt.Errorf("create sync run: %w", err)
	}
	tx.Commit()
	return id, nil
}
func CloseSyncRunSuccess(db *sql.DB, ctx context.Context, syncID uint64, totalSeen uint64, totalDeleted uint64) error {
	log.Logger.Debug().Uint64("sync_id", syncID).Uint64("seen", totalSeen).Uint64("deleted", totalDeleted).Msg("Close sync Success")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncSuccess(ctx, syncID, totalSeen, totalDeleted); err != nil {
		return err
	}
	return tx.Commit()
}
func CloseSyncRunError(db *sql.DB, ctx context.Context, syncID uint64, errorMsg string) error {
	log.Logger.Debug().Uint64("sync_id", syncID).Str("error_msg", errorMsg).Msg("Close sync Success")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.CloseSyncError(ctx, syncID, errorMsg); err != nil {
		return err
	}
	return tx.Commit()
}
