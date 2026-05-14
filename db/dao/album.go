package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"

	"database/sql"
)

const albumFields = `a.id, a.parent_id, a.name, a.description, a.rank, a.ancestor_ids, a.rule_json, a.cover_image_id, a.acl_level, a.acl_user_id, a.updated_at`

const getAlbumById = `SELECT ` + albumFields + ` FROM albums a WHERE a.id=?`
const createAlbum = `INSERT INTO albums (parent_id, name, description, rank, rule_json, acl_level, acl_user_id, ancestor_ids) VALUES (?,?,?,?,?,?,?, JSON_ARRAY())`
const updateAlbum = `UPDATE albums SET parent_id=?, name=?, description=?, rank=?, rule_json=?, acl_level=?, acl_user_id=?, cover_image_id=?  WHERE id=?`
const deleteAlbum = `DELETE FROM albums WHERE id=?`

const queryAlbum = `SELECT ` + albumFields + ` FROM albums as a`

const bindAlbumImage = `INSERT INTO album_images (album_id, image_id, position) VALUES (?,?,?)`
const breakAlbumImage = `DELETE FROM album_images WHERE album_id = ? AND image_id = ?`

// children by parent (parent_id may be NULL)
const queryAlbumByParentACLBegin = `
SELECT ` + albumFields + `
FROM albums a
WHERE `

const queryAlbumByParentACLPagedEnd = `
AND %s 
ORDER BY a.rank ASC, a.name ASC
LIMIT ?, ?
`
const queryAlbumByParentACLPaged = queryAlbumByParentACLBegin +
	` a.parent_id = ? ` + queryAlbumByParentACLPagedEnd
const queryAlbumByParentACLRootPaged = queryAlbumByParentACLBegin +
	` a.parent_id IS NULL ` + queryAlbumByParentACLPagedEnd

const countAlbumByParentACLBegin = `
SELECT count(*)
FROM albums a
WHERE `

const countAlbumByParentACLEnd = `
AND %s 
`
const countAlbumByParentACL = countAlbumByParentACLBegin +
	` a.parent_id = ? ` + countAlbumByParentACLEnd
const countAlbumByParentACLRoot = countAlbumByParentACLBegin +
	` a.parent_id IS NULL ` + countAlbumByParentACLEnd

// reorder siblings
const updateAlbumRank = `UPDATE albums SET rank=? WHERE id=?`

const updateAlbumParentAndRank = `UPDATE albums SET parent_id = ?, rank = ? WHERE id = ?`

const getAlbumMaxRankByParent = `SELECT COALESCE(MAX(rank), -1) FROM albums WHERE parent_id = ?`

const updateAlbumQuick = `UPDATE albums SET name = ?, description = ? WHERE id = ?`

const queryAlbumDescendantIDsByACL = `SELECT a.id
FROM albums a
WHERE JSON_CONTAINS(a.ancestor_ids, ?)
AND %s `

const queryAlbumGraph = `SELECT a.id, a.parent_id, a.name FROM albums AS a`

const updateAlbumParent = `
UPDATE albums a
LEFT JOIN albums p ON p.id = a.parent_id
SET a.ancestor_ids =
    CASE
        WHEN a.parent_id IS NULL THEN JSON_ARRAY(a.id)
        ELSE JSON_ARRAY_APPEND(p.ancestor_ids, '$', a.id)
    END
WHERE a.id = ?;
`

const queryAlbumChildren = `
SELECT a.ID FROM albums AS a WHERE a.parent_id = ?;
`

const queryAlbumsIdByCover = `
SELECT a.ID FROM albums AS a WHERE a.cover_image_id = ?;
`

const countAlbum = `
SELECT COUNT(*) FROM albums;
`

const queryAlbumIDByImageID = `
SELECT ai.album_id FROM album_images ai WHERE ai.image_id = ?;
`
const queryAlbumByImageIDACL = `
SELECT ` + albumFields + ` FROM album_images ai 
JOIN albums a
ON ai.album_id = a.id
WHERE ai.image_id = ? AND %s ;
`

const reorderAllImages = `
UPDATE album_images ai
JOIN (
    SELECT
        ai.album_id,
        ai.image_id,
        ROW_NUMBER() OVER (
            PARTITION BY ai.album_id
            ORDER BY i.order_date, i.filename, i.id
        ) AS new_position
    FROM album_images ai
    JOIN images i ON i.id = ai.image_id
) x
ON ai.album_id = x.album_id
AND ai.image_id = x.image_id
SET ai.position = x.new_position;
`

const countAlbumDescendantIDsByACL = `SELECT count(a.id)
FROM albums a
WHERE JSON_CONTAINS(a.ancestor_ids, ?)
AND %s `

const countAlbumImagesDescendantIDsByACL = `SELECT count(DISTINCT i.id)
FROM albums a
JOIN album_images ai ON a.ID = ai.album_id
JOIN images i ON i.id = ai.image_id
AND %s 
WHERE JSON_CONTAINS(a.ancestor_ids, ?)
AND %s `

const getAlbumByIdACL = `
SELECT ` + albumFields + ` FROM albums a WHERE 
a.id=?
AND %s `

const updateAlbumCountersBydepth = `
UPDATE albums parent
LEFT JOIN (
    SELECT album_id, COUNT(*) AS own_images
    FROM album_images
    GROUP BY album_id
) own ON own.album_id = parent.id

LEFT JOIN (
    SELECT 
        parent_id,
        COUNT(*) AS direct_children,
        SUM(image_count) AS sum_child_images,
        SUM(subalbum_count) AS sum_child_albums
    FROM albums
    WHERE parent_id IS NOT NULL
    GROUP BY parent_id
) ch ON ch.parent_id = parent.id

SET
    parent.image_count =
        IFNULL(own.own_images, 0) + IFNULL(ch.sum_child_images, 0),

    parent.subalbum_count =
        IFNULL(ch.direct_children, 0) + IFNULL(ch.sum_child_albums, 0)

WHERE JSON_LENGTH(parent.ancestor_ids) = ?;
`

func parseAlbum(row *sql.Row) (dbo.Album, error) {
	var a dbo.Album
	var ancestors []byte
	err := row.Scan(&a.ID, &a.ParentID, &a.Name, &a.Description, &a.Rank, &ancestors, &a.RuleJSON, &a.CoverImageID,
		&a.ACLLevel, &a.ACLUserID, &a.UpdatedAt)
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
			&a.ACLLevel, &a.ACLUserID, &a.UpdatedAt)
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
	_, err := q.db.ExecContext(ctx, createAlbum, a.ParentID, a.Name, a.Description, a.Rank, a.RuleJSON, a.ACLLevel, a.ACLUserID)
	return err
}

func (q *Queries) UpdateAlbum(ctx context.Context, a dbo.Album) error {
	_, err := q.db.ExecContext(ctx, updateAlbum, a.ParentID, a.Name, a.Description, a.Rank, a.RuleJSON, a.ACLLevel, a.ACLUserID, a.CoverImageID, a.ID)
	return err
}

func (q *Queries) DeleteAlbum(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteAlbum, id)
	return err
}

func (q *Queries) QueryAlbum(ctx context.Context) ([]dbo.Album, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return parseAlbums(rows)
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
func (q *Queries) QueryAlbumByParentACLPaged(ctx context.Context, parentID *uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Album, error) {
	sql, args := CreateAclWhere("a", acl)
	sqlStr := queryAlbumByParentACLRootPaged
	if parentID != nil {
		sqlStr = queryAlbumByParentACLPaged
		args = append([]any{parentID}, args...)
	}
	args = append(args, from)
	args = append(args, qty)
	sqlStr = fmt.Sprintf(sqlStr, sql)
	rows, err := q.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return parseAlbums(rows)
}

func (q *Queries) CountAlbumByParentACL(ctx context.Context, parentID *uint64, acl dbo.ACLContext) (uint64, error) {
	sql, args := CreateAclWhere("a", acl)
	sqlStr := countAlbumByParentACLRoot
	if parentID != nil {
		sqlStr = countAlbumByParentACL
		args = append([]any{parentID}, args...)
	}

	sqlStr = fmt.Sprintf(sqlStr, sql)
	row := q.db.QueryRowContext(ctx, sqlStr, args...)
	var count uint64
	err := row.Scan(&count)
	return count, err

}

func (q *Queries) UpdateAlbumRank(ctx context.Context, albumID uint64, rank int) error {
	_, err := q.db.ExecContext(ctx, updateAlbumRank, rank, albumID)
	return err
}

func (q *Queries) GetAlbumMaxRankByParent(ctx context.Context, parentID *uint64) (int, error) {
	row := q.db.QueryRowContext(ctx, getAlbumMaxRankByParent, parentID)
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

func (q *Queries) QueryAlbumDescendantIDsByACL(ctx context.Context, rootAlbumID uint64, acl dbo.ACLContext) ([]uint64, error) {

	params := []any{fmt.Sprintf("[%d]", rootAlbumID)}
	sql, aclParams := CreateAclWhere("a", acl)

	params = append(params, aclParams...)

	rows, err := q.db.QueryContext(
		ctx,
		fmt.Sprintf(queryAlbumDescendantIDsByACL, sql), params...)
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

func (q *Queries) QueryAlbumGraph(ctx context.Context) ([]dbo.AlbumGraph, error) {

	rows, err := q.db.QueryContext(ctx, queryAlbumGraph)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := make([]dbo.AlbumGraph, 0)
	for rows.Next() {
		album := dbo.AlbumGraph{}
		if err := rows.Scan(&album.ID, &album.ParentID, &album.Name); err != nil {
			return nil, err
		}
		albums = append(albums, album)
	}
	return albums, rows.Err()
}

func (q *Queries) UpdateAlbumParent(ctx context.Context, albumID uint64) error {
	_, err := q.db.ExecContext(ctx, updateAlbumParent, albumID)
	return err
}

func (q *Queries) QueryAlbumsChildren(ctx context.Context, albumID uint64) ([]uint64, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumChildren, albumID)
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

func (q *Queries) QueryAlbumsIDbyCover(ctx context.Context, cover uint64) ([]uint64, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumsIdByCover, cover)
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

func (q *Queries) CountAlbum(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countAlbum)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) QueryAlbumIDByImageID(ctx context.Context, imageID uint64) ([]uint64, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumIDByImageID, imageID)
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

func (q *Queries) QueryAlbumByImageIDACL(ctx context.Context, imageID uint64, acl dbo.ACLContext) ([]dbo.Album, error) {
	aclWhere, aclParams := CreateAclWhere("a", acl)

	params := []any{imageID}
	params = append(params, aclParams...)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryAlbumByImageIDACL, aclWhere), params...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return parseAlbums(rows)
}

func (q *Queries) ReorderAllImage(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, reorderAllImages)
	return err
}

func (q *Queries) CountAlbumDescendantIDsByACL(ctx context.Context, albumId uint64, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("a", acl)
	params := []any{fmt.Sprintf("[%d]", albumId)}
	params = append(params, aclParams...)
	sql := fmt.Sprintf(countAlbumDescendantIDsByACL, aclWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) CountAlbumImagesDescendantIDsByACL(ctx context.Context, albumID uint64, acl dbo.ACLContext) (uint64, error) {
	aclAWhere, aclAParams := CreateAclWhere("a", acl)
	aclIWhere, aclIParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclIParams...)
	params = append(params, fmt.Sprintf("[%d]", albumID))
	params = append(params, aclAParams...)
	sql := fmt.Sprintf(countAlbumImagesDescendantIDsByACL, aclIWhere, aclAWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetAlbumByIdACL(ctx context.Context, albumID uint64, acl dbo.ACLContext) (dbo.Album, error) {
	aclWhere, aclParams := CreateAclWhere("a", acl)
	params := []any{albumID}
	params = append(params, aclParams...)
	sql := fmt.Sprintf(getAlbumByIdACL, aclWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	return parseAlbum(row)
}

//
// =========================================================
// Public API functions
// =========================================================
//

func BindAlbumImage(db *sql.DB, c context.Context, albumId, imageId uint64, pos *uint32) error {
	logScope, ctx := logging.Enter(c, "dao/album/bindImage", imageId, map[string]any{"album_id": albumId, "image_id": imageId})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindAlbumImage(ctx, albumId, imageId, pos); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

func BreakAlbumImage(db *sql.DB, c context.Context, albumId uint64, imageId uint64) error {
	logScope, ctx := logging.Enter(c, "dao/album/breakImage", imageId, map[string]any{"album_id": albumId, "image_id": imageId})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.BreakAlbumImage(ctx, albumId, imageId); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

func GetAlbumById(db *sql.DB, c context.Context, albumID uint64) (dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/get/ById", albumID, map[string]any{"album_id": albumID})
	q := NewQueries(db)
	a, err := q.GetAlbumById(ctx, albumID)
	return a, returnWrapNotFound(logScope, err, "album")
}

func updateAlbumParentRecursive(q *Queries, c context.Context, albumID uint64) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/parent/recursive", albumID, map[string]any{"album_id": albumID})
	queue := []uint64{albumID}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		err := q.UpdateAlbumParent(ctx, id)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		children, err := q.QueryAlbumsChildren(ctx, id)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		queue = append(queue, children...)
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

func CreateAlbum(db *sql.DB, c context.Context, a *dbo.Album) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/create", a.Name, map[string]any{"name": a.Name,
		"album": a})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	if err := q.CreateAlbum(ctx, *a); err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}

	id, err := q.GetLastId(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	a.ID = utils.PtrUint64(id)
	logging.Debug(logScope, "after inser", map[string]any{"id": id})
	err = updateAlbumParentRecursive(q, ctx, id)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		logging.ExitErr(logScope, err)
	} else {
		logging.Exit(logScope, "ok", map[string]any{"new_id": id})
	}
	return uint64(id), err
}

func UpdateAlbum(db *sql.DB, c context.Context, a dbo.Album) error {
	logScope, ctx := logging.Enter(c, "dao/album/update", a.ID, map[string]any{"album": a})
	if a.ID == nil {
		err := sql.ErrNoRows
		logging.ExitErr(logScope, err)
		return err
	}
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	if err := q.UpdateAlbum(ctx, a); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	err = updateAlbumParentRecursive(q, ctx, *a.ID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

func DeleteAlbum(db *sql.DB, c context.Context, id uint64) error {
	logScope, ctx := logging.Enter(c, "dao/album/delete", id, map[string]any{"id": id})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteAlbum(ctx, id); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

func QueryAlbum(db *sql.DB, c context.Context) ([]dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query", nil, nil)
	q := NewQueries(db)
	albums, err := q.QueryAlbum(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}

func QueryAlbumByParentACLPaged(db *sql.DB, c context.Context, parentID *uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/byParent/ACL/Paged", parentID, map[string]any{
		"partent": parentID,
		"ACL":     acl,
		"from":    from,
		"qty":     qty,
	})
	q := NewQueries(db)
	albums, err := q.QueryAlbumByParentACLPaged(ctx, parentID, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}

func CountAlbumByParentACL(db *sql.DB, c context.Context, parentID *uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/byParent/ACL", parentID, map[string]any{
		"partent": parentID,
		"ACL":     acl,
	})
	q := NewQueries(db)
	count, err := q.CountAlbumByParentACL(ctx, parentID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", nil)
	return count, nil
}

func ReorderAlbumsSibling(db *sql.DB, c context.Context, albumIDs []uint64) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/reorder", nil, map[string]any{
		"ids": albumIDs})
	if len(albumIDs) == 0 {
		logging.Exit(logScope, "emply list", nil)
		return nil
	}

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	// use dense ranks: 0,1,2,... according to slice order
	for rank, id := range albumIDs {
		if err := q.UpdateAlbumRank(ctx, id, rank); err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
	}

	return logging.Return(logScope, tx.Commit())
}

func MoveAlbum(db *sql.DB, c context.Context, albumID uint64, newParentID *uint64) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/move", albumID, map[string]any{
		"album_id":   albumID,
		"new_parent": newParentID,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	maxRank, err := q.GetAlbumMaxRankByParent(ctx, newParentID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	newRank := maxRank + 1

	if err := q.UpdateAlbumParentAndRank(ctx, albumID, newParentID, newRank); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}
func UpdateAlbumQuick(db *sql.DB, c context.Context, id uint64, name string, description *string) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/quick", id, map[string]any{
		"album_id": id,
	})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateAlbumQuick(ctx, id, name, description); err != nil {
		logging.ExitErr(logScope, err)
	}

	return logging.Return(logScope, tx.Commit())
}

func CollectAlbumSubtreeIDs(db *sql.DB, c context.Context, rootAlbumID uint64, acl dbo.ACLContext) ([]uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/id/subtree/ACL", rootAlbumID, map[string]any{
		"album_id": rootAlbumID,
		"acl":      acl,
	})

	q := NewQueries(db)
	albums, err := q.QueryAlbumDescendantIDsByACL(ctx, rootAlbumID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
	} else {
		logging.Exit(logScope, "ok", map[string]any{
			"albums": albums,
		})
	}
	return albums, err
}

func CountAlbumSubtree(db *sql.DB, c context.Context, rootAlbumID uint64, acl dbo.ACLContext) (int, int, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/subtree/ACL", rootAlbumID, map[string]any{
		"album_id": rootAlbumID,
		"acl":      acl,
	})

	q := NewQueries(db)

	albumIDs, err := q.QueryAlbumDescendantIDsByACL(ctx, rootAlbumID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, 0, err
	}

	if len(albumIDs) == 0 {
		logging.Exit(logScope, "ok", map[string]any{
			"albums": 0,
			"images": 0,
		})
		return 0, 0, nil
	}

	childAlbumCount := len(albumIDs) - 1
	if childAlbumCount < 0 {
		childAlbumCount = 0
	}
	imageCount, err := q.CountImageByAlbums(ctx, albumIDs, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"albums": childAlbumCount,
		"images": imageCount,
	})
	return childAlbumCount, int(imageCount), nil

}
func QueryAlbumGraph(db *sql.DB, c context.Context) ([]dbo.AlbumGraph, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/AlbumGraph", nil, nil)
	q := NewQueries(db)
	albums, err := q.QueryAlbumGraph(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}
func QueryAlbumsIdByCover(db *sql.DB, c context.Context, cover uint64) ([]uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/id/byCover", cover, map[string]any{
		"cover": cover,
	})
	q := NewQueries(db)
	albums, err := q.QueryAlbumsIDbyCover(ctx, cover)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}

func CountAlbum(db *sql.DB, c context.Context) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count", nil, nil)
	q := NewQueries(db)
	qty, err := q.CountAlbum(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}
func QueryAlbumIDByImageID(db *sql.DB, c context.Context, imageID uint64) ([]uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/id/byImage", imageID, map[string]any{
		"image_id": imageID,
	})
	q := NewQueries(db)
	albums, err := q.QueryAlbumIDByImageID(ctx, imageID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}
func QueryAlbumsIDImageIDACL(db *sql.DB, c context.Context, imageID uint64, acl dbo.ACLContext) ([]dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/query/byImage/byACL", imageID, map[string]any{
		"image_id": imageID,
		"ac":       acl,
	})
	q := NewQueries(db)
	albums, err := q.QueryAlbumByImageIDACL(ctx, imageID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{"found": len(albums)})
	return albums, nil
}

func ReorderAllImages(db *sql.DB, c context.Context) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/image/reorder", nil, nil)
	q := NewQueries(db)
	err := q.ReorderAllImage(ctx)
	return logging.Return(logScope, err)
}

func CountAlbumDescendantIDsByACL(db *sql.DB, c context.Context, albumId uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/albums/descendant/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumId,
	})
	q := NewQueries(db)
	qty, err := q.CountAlbumDescendantIDsByACL(ctx, albumId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

func CountAlbumImagesDescendantIDsByACL(db *sql.DB, c context.Context, albumId uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/images/descendant/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumId,
	})
	q := NewQueries(db)
	qty, err := q.CountAlbumImagesDescendantIDsByACL(ctx, albumId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}
func GetAlbumByIDACL(db *sql.DB, c context.Context, albumId uint64, acl dbo.ACLContext) (dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/get/byID/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumId,
	})
	q := NewQueries(db)
	album, err := q.GetAlbumByIdACL(ctx, albumId, acl)
	return album, returnWrapNotFound(logScope, err, "album")
}
