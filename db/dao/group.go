package dao

import (
	"context"

	"database/sql"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/rs/zerolog/log"
)

const groupFields = `g.id, g.name`

const getGroupById = `SELECT ` + groupFields + ` FROM groups g WHERE g.id=?`
const createGroup = `INSERT INTO groups (name) VALUES (?)`
const deleteGroup = `DELETE FROM groups WHERE id=?`

func parseGroup(row *sql.Row) (dbo.Group, error) {
	var g dbo.Group
	err := row.Scan(&g.ID, &g.Name)
	return g, err
}

const bindGroupUser = `INSERT INTO user_groups (user_id, group_id) VALUES (?,?)`

func (q *Queries) BindGroupUser(ctx context.Context, groupId, userId uint64) error {
	_, err := q.db.ExecContext(ctx, bindGroupUser, userId, groupId)
	return err
}

const breakGroupUser = `DELETE FROM user_groups WHERE user_id = ? AND group_id = ?`

func (q *Queries) BreakGroupUser(ctx context.Context, groupId, userId uint64) error {
	_, err := q.db.ExecContext(ctx, breakGroupUser, userId, groupId)
	return err
}

func (q *Queries) GetGroupById(ctx context.Context, id uint64) (dbo.Group, error) {
	row := q.db.QueryRowContext(ctx, getGroupById, id)
	return parseGroup(row)
}

func (q *Queries) CreateGroup(ctx context.Context, name string) error {
	_, err := q.db.ExecContext(ctx, createGroup, name)
	return err
}

func (q *Queries) DeleteGroup(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteGroup, id)
	return err
}

func BindGroupUser(db *sql.DB, ctx context.Context, groupId, userId uint64) error {
	log.Logger.Debug().Uint64("group", groupId).Uint64("user", userId).Msg("Bind Group User")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindGroupUser(ctx, groupId, userId); err != nil {
		return err
	}
	return tx.Commit()
}
func CreateGroup(db *sql.DB, ctx context.Context, name string) error {
	log.Logger.Debug().Str("group", name).Msg("Create Group")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateGroup(ctx, name); err != nil {
		return err
	}
	return tx.Commit()
}

func DeleteGroup(db *sql.DB, ctx context.Context, id uint64) error {
	log.Logger.Debug().Uint64("group", id).Msg("Delete Group")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteGroup(ctx, id); err != nil {
		return err
	}
	return tx.Commit()
}
