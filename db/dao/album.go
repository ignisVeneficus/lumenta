package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"

	"database/sql"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const albumFields = `a.id, a.parent_id, a.name, a.description, a.rank, a.ancestor_ids, a.rule_json, a.cover_image_id, a.child_album_count, a.image_count, a.acl_scope, a.acl_user_id, a.acl_group_id, a.updated_at`

const getAlbumById = `SELECT ` + albumFields + ` FROM albums a WHERE a.id=?`
const createAlbum = `INSERT INTO albums (parent_id, name, description, rank, ancestor_ids, rule_json, acl_scope, acl_user_id, acl_group_id) VALUES (?,?,?,?,?,?,?,?)`
const updateAlbum = `UPDATE albums SET name=?, description=?, rank=?, rule_json=?, acl_scope=?, acl_user_id=?, acl_group_id=? WHERE id=?`
const deleteAlbum = `DELETE FROM albums WHERE id=?`

const bindAlbumImage = `INSERT INTO album_images (album_id, image_id, position) VALUES (?,?,?)`
const breakAlbumImage = `DELETE FROM album_images WHERE album_id = ? AND image_id = ?`

// children by parent (parent_id may be NULL)
const queryChildAlbumsByParentACLBegin = `
SELECT ` + albumFields + `
FROM albums a
WHERE `

const queryChildAlbumsByParentACLEnd = `
AND ` + aclWhereClause + `
ORDER BY a.rank ASC, a.name ASC
`
const queryChildAlbumsByParentACL = queryChildAlbumsByParentACLBegin +
	` a.parent_id = ? ` + queryChildAlbumsByParentACLEnd
const queryChildAlbumsByParentACLRoot = queryChildAlbumsByParentACLBegin +
	` a.parent_id IS NULL ` + queryChildAlbumsByParentACLEnd

// reorder siblings
const updateAlbumRank = `UPDATE albums SET rank=? WHERE id=?`

const updateAlbumParentAndRank = `UPDATE albums SET parent_id = ?, rank = ? WHERE id = ?`

const getMaxRankByParent = `SELECT COALESCE(MAX(rank), -1) FROM albums WHERE parent_id = ?`

const updateAlbumQuick = `UPDATE albums SET name = ?, description = ? WHERE id = ?`

const queryDescendantAlbumIDsByACL = `SELECT a.id
FROM albums a
WHERE JSON_CONTAINS(a.ancestor_ids, CAST(? AS JSON))
` + aclWhereClause

func parseAlbum(row *sql.Row) (dbo.Album, error) {
	var a dbo.Album
	var ancestors []byte
	err := row.Scan(&a.ID, &a.ParentID, &a.Name, &a.Description, &a.Rank, &ancestors, &a.RuleJSON, &a.CoverImageID,
		&a.ChildAlbumCount, &a.ImageCount, &a.ACLScope, &a.ACLUserID, &a.ACLGroupID, &a.UpdatedAt)
	if err != nil {
		return a, err
	}
	err = json.Unmarshal(ancestors, &a.AncestorIDs)
	return a, err
}

func parseAlbums(rows *sql.Rows) ([]dbo.Album, error) {
	out := make([]dbo.Album, 0)
	for rows.Next() {
		var a dbo.Album
		var ancestors []byte
		err := rows.Scan(&a.ID, &a.ParentID, &a.Name, &a.Description, &a.Rank, &ancestors, &a.RuleJSON, &a.CoverImageID,
			&a.ChildAlbumCount, &a.ImageCount, &a.ACLScope, &a.ACLUserID, &a.ACLGroupID, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(ancestors, &a.AncestorIDs); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (q *Queries) GetAlbumById(ctx context.Context, id uint64) (dbo.Album, error) {
	row := q.db.QueryRowContext(ctx, getAlbumById, id)
	return parseAlbum(row)
}

func (q *Queries) CreateAlbum(ctx context.Context, a dbo.Album) error {
	anc, _ := json.Marshal(a.AncestorIDs)
	_, err := q.db.ExecContext(ctx, createAlbum, a.ParentID, a.Name, a.Description, a.Rank, anc, a.RuleJSON, a.ACLScope, a.ACLUserID, a.ACLGroupID)
	return err
}

func (q *Queries) UpdateAlbum(ctx context.Context, a dbo.Album) error {
	_, err := q.db.ExecContext(ctx, updateAlbum, a.Name, a.Description, a.Rank, a.RuleJSON, a.ACLScope, a.ACLUserID, a.ACLGroupID, a.ID)
	return err
}

func (q *Queries) DeleteAlbum(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteAlbum, id)
	return err
}

func (q *Queries) BindAlbumImage(ctx context.Context, albumId, imageId uint64, pos *uint32) error {
	_, err := q.db.ExecContext(ctx, bindAlbumImage, albumId, imageId, pos)
	return err
}

func (q *Queries) BreakAlbumImage(ctx context.Context, albumId, imageId uint64) error {
	_, err := q.db.ExecContext(ctx, breakAlbumImage, albumId, imageId)
	return err
}

// parentID may be nil (root level)
func (q *Queries) QueryChildAlbumsByParentACL(ctx context.Context, parentID *uint64, acl ACLContext) ([]dbo.Album, error) {
	args := acl.AsParamArray()
	sqlStr := queryChildAlbumsByParentACLRoot
	if parentID != nil {
		sqlStr = queryChildAlbumsByParentACL
		args = append([]any{parentID}, args...)
	}
	rows, err := q.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseAlbums(rows)
}

func (q *Queries) UpdateAlbumRank(ctx context.Context, albumID uint64, rank int) error {
	_, err := q.db.ExecContext(ctx, updateAlbumRank, rank, albumID)
	return err
}

func (q *Queries) GetMaxAlbumRankByParent(ctx context.Context, parentID *uint64) (int, error) {
	row := q.db.QueryRowContext(ctx, getMaxRankByParent, parentID)
	var max int
	err := row.Scan(&max)
	return max, err
}

func (q *Queries) UpdateAlbumParentAndRank(ctx context.Context, albumID uint64, parentID *uint64, rank int) error {
	_, err := q.db.ExecContext(ctx, updateAlbumParentAndRank, parentID, rank, albumID)
	return err
}

func (q *Queries) UpdateAlbumQuick(ctx context.Context, id uint64, name string, description *string) error {
	_, err := q.db.ExecContext(ctx, updateAlbumQuick, name, description, id)
	return err
}

func (q *Queries) QueryDescendantAlbumIDsByACL(ctx context.Context, rootAlbumID uint64, acl ACLContext) ([]uint64, error) {

	rootJSON := fmt.Sprintf("[%d]", rootAlbumID)

	rows, err := q.db.QueryContext(
		ctx,
		queryDescendantAlbumIDsByACL,
		rootJSON,
		acl.IsAnyUser(),
		acl.ViewerUserID,
		acl.IsAdmin(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uint64, 0)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func BindAlbumImage(db *sql.DB, ctx context.Context, albumId, imageId uint64, pos *uint32) error {
	log.Logger.Debug().Uint64("album", albumId).Uint64("image", imageId).Msg("Bind Album Image")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindAlbumImage(ctx, albumId, imageId, pos); err != nil {
		return err
	}
	return tx.Commit()
}

func BreakAlbumImage(db *sql.DB, ctx context.Context, albumId uint64, imageId uint64) error {
	log.Logger.Debug().Uint64("album", albumId).Uint64("image", imageId).Msg("Break Album Image")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.BreakAlbumImage(ctx, albumId, imageId); err != nil {
		return err
	}
	return tx.Commit()
}

func GetAlbumById(db *sql.DB, ctx context.Context, id uint64) (dbo.Album, error) {
	log.Logger.Debug().Uint64("album", id).Msg("Get Album")
	q := NewQueries(db)
	a, err := q.GetAlbumById(ctx, id)
	return a, wrapNotFound(err, "album")
}

func CreateAlbum(db *sql.DB, ctx context.Context, a *dbo.Album) (uint64, error) {
	log.Logger.Debug().Str("name", a.Name).Msg("Create Album")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	if err := q.CreateAlbum(ctx, *a); err != nil {
		return 0, err
	}

	id, err := q.GetLastId(ctx)
	if err != nil {
		return 0, err
	}
	a.ID = ptrUint64(uint64(id))

	return uint64(id), tx.Commit()
}

func UpdateAlbum(db *sql.DB, ctx context.Context, a dbo.Album) error {
	log.Logger.Debug().Uint64("album", *a.ID).Msg("Update Album")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateAlbum(ctx, a); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteAlbum(db *sql.DB, ctx context.Context, id uint64) error {
	log.Logger.Debug().Uint64("album", id).Msg("Delete Album")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteAlbum(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}

func QueryChildAlbumsByParentACL(db *sql.DB, ctx context.Context, parentID *uint64, acl ACLContext) ([]dbo.Album, error) {
	log.Logger.Debug().
		Interface("parent", parentID).
		Object("acl", logging.WithLevel(zerolog.DebugLevel, &acl)).
		Msg("Query Child Albums By Parent + ACL")

	q := NewQueries(db)
	albums, err := q.QueryChildAlbumsByParentACL(ctx, parentID, acl)
	if err != nil {
		return nil, err
	}
	return albums, nil
}

func ReorderSiblingAlbums(db *sql.DB, ctx context.Context, albumIDs []uint64) error {

	log.Logger.Debug().
		Int("count", len(albumIDs)).
		Msg("Reorder Sibling Albums")

	if len(albumIDs) == 0 {
		return nil
	}

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	// use dense ranks: 0,1,2,... according to slice order
	for rank, id := range albumIDs {
		if err := q.UpdateAlbumRank(ctx, id, rank); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func MoveAlbum(db *sql.DB, ctx context.Context, albumID uint64, newParentID *uint64) error {
	log.Logger.Debug().Uint64("album", albumID).Interface("new_parent", newParentID).Msg("Move Album")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	maxRank, err := q.GetMaxAlbumRankByParent(ctx, newParentID)
	if err != nil {
		return err
	}

	newRank := maxRank + 1

	if err := q.UpdateAlbumParentAndRank(ctx, albumID, newParentID, newRank); err != nil {
		return err
	}

	return tx.Commit()
}
func UpdateAlbumQuick(db *sql.DB, ctx context.Context, id uint64, name string, description *string) error {
	log.Logger.Debug().Uint64("album", id).Msg("Quick Update Album")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateAlbumQuick(ctx, id, name, description); err != nil {
		return err
	}

	return tx.Commit()
}

func CollectAlbumSubtreeIDs(db *sql.DB, ctx context.Context, rootAlbumID uint64, acl ACLContext) ([]uint64, error) {
	log.Logger.Debug().Uint64("album", rootAlbumID).Object("acl", logging.WithLevel(zerolog.DebugLevel, &acl)).Msg("Count Album Subtree IDs")

	q := NewQueries(db)
	return q.QueryDescendantAlbumIDsByACL(
		ctx,
		rootAlbumID,
		acl,
	)
}

func CountAlbumSubtree(db *sql.DB, ctx context.Context, rootAlbumID uint64, acl ACLContext) (int, int, error) {
	log.Logger.Debug().Uint64("album", rootAlbumID).Object("acl", logging.WithLevel(zerolog.DebugLevel, &acl)).Msg("Count Album Subtree")

	q := NewQueries(db)

	// 1️⃣ rekurzív album ID lista (root is benne van)
	albumIDs, err := q.QueryDescendantAlbumIDsByACL(
		ctx,
		rootAlbumID,
		acl,
	)
	if err != nil {
		return 0, 0, err
	}

	if len(albumIDs) == 0 {
		return 0, 0, nil
	}

	childAlbumCount := len(albumIDs) - 1
	if childAlbumCount < 0 {
		childAlbumCount = 0
	}
	imageCount, err := q.CountImagesByAlbums(ctx, albumIDs, acl)
	if err != nil {
		return 0, 0, nil
	}

	return childAlbumCount, int(imageCount), nil

}
