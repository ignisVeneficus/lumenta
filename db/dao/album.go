package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	"database/sql"
)

const albumFields = `a.id, a.parent_id, a.name, a.description, a.rank, a.ancestor_ids, a.rule_json, a.cover_image_id, a.acl_level, a.acl_user_id, a.updated_at`

const getAlbumByID = `SELECT ` + albumFields + ` FROM albums a WHERE a.id=?`
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

const getAlbumByIDACL = `
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

// parseAlbum scans a single album row into an Album value.
//
// Input:
//   - row: SQL row containing the albumFields columns.
//
// Output:
//   - dbo.Album: scanned album with decoded ancestor IDs.
//   - error: scan or JSON decode error, if any.
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

// parseAlbums scans album rows into Album values.
//
// Input:
//   - rows: SQL rows containing the albumFields columns.
//
// Output:
//   - []dbo.Album: scanned albums with decoded ancestor IDs.
//   - error: scan, JSON decode, or row iteration error.
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

// GetAlbumByID reads an album by ID.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to read.
//
// Output:
//   - dbo.Album: matching album.
//   - error: query, scan, decode, or sql.ErrNoRows error.
func (q *Queries) GetAlbumByID(ctx context.Context, albumID dbo.AlbumID) (dbo.Album, error) {
	row := q.db.QueryRowContext(ctx, getAlbumByID, albumID)
	return parseAlbum(row)
}

// CreateAlbum inserts an album.
//
// Input:
//   - ctx: request context.
//   - a: album data to insert.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) CreateAlbum(ctx context.Context, a dbo.Album) error {
	_, err := q.db.ExecContext(ctx, createAlbum, a.ParentID, a.Name, a.Description, a.Rank, a.RuleJSON, a.ACLLevel, a.ACLUserID)
	return err
}

// UpdateAlbum updates an album row.
//
// Input:
//   - ctx: request context.
//   - a: album data to write; a.ID identifies the row.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateAlbum(ctx context.Context, a dbo.Album) error {
	_, err := q.db.ExecContext(ctx, updateAlbum, a.ParentID, a.Name, a.Description, a.Rank, a.RuleJSON, a.ACLLevel, a.ACLUserID, a.CoverImageID, a.ID)
	return err
}

// DeleteAlbum deletes an album by ID.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to delete.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) DeleteAlbum(ctx context.Context, albumID dbo.AlbumID) error {
	_, err := q.db.ExecContext(ctx, deleteAlbum, albumID)
	return err
}

// QueryAlbum reads all albums.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - []dbo.Album: albums returned by the query.
//   - error: query, scan, decode, or row iteration error.
func (q *Queries) QueryAlbum(ctx context.Context) ([]dbo.Album, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return parseAlbums(rows)
}

// BindAlbumImage creates an album-image relation.
//
// Input:
//   - ctx: request context.
//   - albumID: album to bind.
//   - imageID: image to bind.
//   - pos: relation position
//
// Output:
//   - error: exec error, if any.
func (q *Queries) BindAlbumImage(ctx context.Context, albumID dbo.AlbumID, imageID dbo.ImageID, pos *uint32) error {
	_, err := q.db.ExecContext(ctx, bindAlbumImage, albumID, imageID, pos)
	return err
}

// BreakAlbumImage removes an album-image relation.
//
// Input:
//   - ctx: request context.
//   - albumID: album in the relation.
//   - imageID: image in the relation.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) BreakAlbumImage(ctx context.Context, albumID dbo.AlbumID, imageID dbo.ImageID) error {
	_, err := q.db.ExecContext(ctx, breakAlbumImage, albumID, imageID)
	return err
}

// QueryAlbumByParentACLPaged reads child albums filtered by parent and ACL.
//
// Input:
//   - ctx: request context.
//   - parentID: parent album ID; nil means root level.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Album: matching albums.
//   - error: query, scan, decode, or row iteration error.
func (q *Queries) QueryAlbumByParentACLPaged(ctx context.Context, parentID *dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Album, error) {
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

// CountAlbumByParentACL counts child albums filtered by parent and ACL.
//
// Input:
//   - ctx: request context.
//   - parentID: parent album ID; nil means root level.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching album count.
//   - error: query or scan error, if any.
func (q *Queries) CountAlbumByParentACL(ctx context.Context, parentID *dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
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

// UpdateAlbumRank updates an album sibling rank.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to update.
//   - rank: new rank value.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateAlbumRank(ctx context.Context, albumID dbo.AlbumID, rank int) error {
	_, err := q.db.ExecContext(ctx, updateAlbumRank, rank, albumID)
	return err
}

// GetAlbumMaxRankByParent reads the maximum rank under a parent album.
//
// Input:
//   - ctx: request context.
//   - parentID: parent album ID
//
// Output:
//   - int: maximum rank, or -1 when the SQL aggregate has no rows.
//   - error: query or scan error, if any.
func (q *Queries) GetAlbumMaxRankByParent(ctx context.Context, parentID *dbo.AlbumID) (int, error) {
	row := q.db.QueryRowContext(ctx, getAlbumMaxRankByParent, parentID)
	var max int
	err := row.Scan(&max)
	return max, err
}

// UpdateAlbumParentAndRank updates an album parent and rank.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to update.
//   - parentID: new parent album ID; nil means root level.
//   - rank: new rank value.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateAlbumParentAndRank(ctx context.Context, albumID dbo.AlbumID, parentID *dbo.AlbumID, rank int) error {
	_, err := q.db.ExecContext(ctx, updateAlbumParentAndRank, parentID, rank, albumID)
	return err
}

// UpdateAlbumQuick updates an album name and description.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to update.
//   - name: new album name.
//   - description: new album description.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateAlbumQuick(ctx context.Context, albumID dbo.AlbumID, name string, description *string) error {
	_, err := q.db.ExecContext(ctx, updateAlbumQuick, name, description, albumID)
	return err
}

// QueryAlbumDescendantIDsByACL reads descendant album IDs visible through ACL.
//
// Input:
//   - ctx: request context.
//   - rootAlbumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.AlbumID: matching album IDs, including root album.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryAlbumDescendantIDsByACL(ctx context.Context, rootAlbumID dbo.AlbumID, acl dbo.ACLContext) ([]dbo.AlbumID, error) {

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

	ids := make([]dbo.AlbumID, 0)
	for rows.Next() {
		var id dbo.AlbumID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// QueryAlbumGraph reads album graph nodes.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - []dbo.AlbumGraph: album graph nodes with ID, parent ID, and name.
//   - error: query, scan, or row iteration error.
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

// UpdateAlbumParent recalculates ancestor IDs for one album.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to recalculate.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateAlbumParent(ctx context.Context, albumID dbo.AlbumID) error {
	_, err := q.db.ExecContext(ctx, updateAlbumParent, albumID)
	return err
}

// QueryAlbumsChildren reads direct child album IDs.
//
// Input:
//   - ctx: request context.
//   - albumID: parent album ID.
//
// Output:
//   - []dbo.AlbumID: direct child album IDs.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryAlbumsChildren(ctx context.Context, albumID dbo.AlbumID) ([]dbo.AlbumID, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumChildren, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]dbo.AlbumID, 0)
	for rows.Next() {
		var id dbo.AlbumID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// QueryAlbumsIDbyCover reads album IDs that use an image as cover.
//
// Input:
//   - ctx: request context.
//   - cover: cover image ID.
//
// Output:
//   - []dbo.AlbumID: album IDs using the cover image.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryAlbumsIDbyCover(ctx context.Context, cover dbo.ImageID) ([]dbo.AlbumID, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumsIdByCover, cover)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]dbo.AlbumID, 0)
	for rows.Next() {
		var id dbo.AlbumID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()

}

// CountAlbum counts all albums.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - uint64: album count.
//   - error: query or scan error, if any.
func (q *Queries) CountAlbum(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countAlbum)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// QueryAlbumIDByImageID reads album IDs containing an image.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to look up.
//
// Output:
//   - []dbo.AlbumID: album IDs containing the image.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryAlbumIDByImageID(ctx context.Context, imageID dbo.ImageID) ([]dbo.AlbumID, error) {
	rows, err := q.db.QueryContext(ctx, queryAlbumIDByImageID, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]dbo.AlbumID, 0)
	for rows.Next() {
		var id dbo.AlbumID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// QueryAlbumByImageIDACL reads albums containing an image and visible through ACL.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to look up.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.Album: matching albums.
//   - error: query, scan, decode, or row iteration error.
func (q *Queries) QueryAlbumByImageIDACL(ctx context.Context, imageID dbo.ImageID, acl dbo.ACLContext) ([]dbo.Album, error) {
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

// ReorderAllImage recalculates album-image positions.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) ReorderAllImage(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, reorderAllImages)
	return err
}

// CountAlbumDescendantIDsByACL counts descendant albums visible through ACL.
//
// Input:
//   - ctx: request context.
//   - albumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching album count.
//   - error: query or scan error, if any.
func (q *Queries) CountAlbumDescendantIDsByACL(ctx context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("a", acl)
	params := []any{fmt.Sprintf("[%d]", albumID)}
	params = append(params, aclParams...)
	sql := fmt.Sprintf(countAlbumDescendantIDsByACL, aclWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// GetAlbumByIDACL reads an album by ID when visible through ACL.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to read.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.Album: matching album.
//   - error: query, scan, decode, or sql.ErrNoRows error.
func (q *Queries) GetAlbumByIDACL(ctx context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (dbo.Album, error) {
	aclWhere, aclParams := CreateAclWhere("a", acl)
	params := []any{albumID}
	params = append(params, aclParams...)
	sql := fmt.Sprintf(getAlbumByIDACL, aclWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	return parseAlbum(row)
}

//
// =========================================================
// Public API functions
// =========================================================
//

// BindAlbumImage creates an album-image relation in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album to bind.
//   - imageID: image to bind.
//   - pos: relation position
//
// Output:
//   - error: transaction, bind, or commit error.
func BindAlbumImage(db *sql.DB, c context.Context, albumID dbo.AlbumID, imageID dbo.ImageID, pos *uint32) error {
	logScope, ctx := logging.Enter(c, "dao/album/bindImage", imageID, map[string]any{"album_id": albumID, "image_id": imageID})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindAlbumImage(ctx, albumID, imageID, pos); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

// BreakAlbumImage removes an album-image relation in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album in the relation.
//   - imageID: image in the relation.
//
// Output:
//   - error: transaction, delete, or commit error.
func BreakAlbumImage(db *sql.DB, c context.Context, albumID dbo.AlbumID, imageID dbo.ImageID) error {
	logScope, ctx := logging.Enter(c, "dao/album/breakImage", imageID, map[string]any{"album_id": albumID, "image_id": imageID})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	if err := q.BreakAlbumImage(ctx, albumID, imageID); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

// GetAlbumByID reads an album by ID with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to read.
//
// Output:
//   - dbo.Album: matching album.
//   - error: wrapped not-found, query, scan, or decode error.
func GetAlbumByID(db *sql.DB, c context.Context, albumID dbo.AlbumID) (dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/get/ByID", albumID, map[string]any{"album_id": albumID})
	q := NewQueries(db)
	a, err := q.GetAlbumByID(ctx, albumID)
	return a, returnWrapNotFound(logScope, err, "album")
}

// updateAlbumParentRecursive recalculates ancestor IDs for an album subtree.
//
// Input:
//   - q: query runner.
//   - c: request context.
//   - albumID: root album ID to start from.
//
// Output:
//   - error: update or child-query error, if any.
func updateAlbumParentRecursive(q *Queries, c context.Context, albumID dbo.AlbumID) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/parent/recursive", albumID, map[string]any{"album_id": albumID})
	queue := []dbo.AlbumID{albumID}

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

// CreateAlbum inserts an album, stores its ID on a, and recalculates ancestor IDs.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - a: album to insert; receives the new ID.
//
// Output:
//   - dbo.AlbumID: created album ID.
//   - error: transaction, insert, ID lookup, ancestor update, or commit error.
func CreateAlbum(db *sql.DB, c context.Context, a *dbo.Album) (dbo.AlbumID, error) {
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
	newID := dbo.AlbumID(id)
	a.ID = &newID
	logging.Debug(logScope, "after inser", map[string]any{"id": id})
	err = updateAlbumParentRecursive(q, ctx, newID)
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
	return newID, err
}

// UpdateAlbum updates an album and recalculates ancestor IDs for its subtree.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - a: album data to update; a.ID identifies the row.
//
// Output:
//   - error: sql.ErrNoRows for nil ID, transaction, update, ancestor update, or commit error.
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

// DeleteAlbum deletes an album in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to delete.
//
// Output:
//   - error: transaction, delete, or commit error.
func DeleteAlbum(db *sql.DB, c context.Context, albumID dbo.AlbumID) error {
	logScope, ctx := logging.Enter(c, "dao/album/delete", albumID, map[string]any{"id": albumID})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteAlbum(ctx, albumID); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

// QueryAlbum reads all albums with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - []dbo.Album: albums returned by the query.
//   - error: query, scan, decode, or row iteration error.
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

// QueryAlbumByParentACLPaged reads child albums filtered by parent and ACL with paging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - parentID: parent album ID; nil means root level.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Album: matching albums.
//   - error: query, scan, decode, or row iteration error.
func QueryAlbumByParentACLPaged(db *sql.DB, c context.Context, parentID *dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Album, error) {
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

// CountAlbumByParentACL counts child albums filtered by parent and ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - parentID: parent album ID; nil means root level.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching album count.
//   - error: query or scan error, if any.
func CountAlbumByParentACL(db *sql.DB, c context.Context, parentID *dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
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

// ReorderAlbumsSibling rewrites sibling album ranks from the supplied ID order.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumIDs: album IDs in desired rank order.
//
// Output:
//   - error: transaction, rank update, or commit error.
func ReorderAlbumsSibling(db *sql.DB, c context.Context, albumIDs []dbo.AlbumID) error {
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

// MoveAlbum moves an album under a new parent and appends it after existing siblings.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to move.
//   - newParentID: new parent album ID; nil means root level.
//
// Output:
//   - error: transaction, rank lookup, update, or commit error.
func MoveAlbum(db *sql.DB, c context.Context, albumID dbo.AlbumID, newParentID *dbo.AlbumID) error {
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

// UpdateAlbumQuick updates an album name and description in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to update.
//   - name: new album name.
//   - description: new album description.
//
// Output:
//   - error: transaction, update, or commit error.
func UpdateAlbumQuick(db *sql.DB, c context.Context, albumID dbo.AlbumID, name string, description *string) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/quick", albumID, map[string]any{
		"album_id": albumID,
	})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateAlbumQuick(ctx, albumID, name, description); err != nil {
		logging.ExitErr(logScope, err)
	}

	return logging.Return(logScope, tx.Commit())
}

// CollectAlbumSubtreeIDs reads album IDs in a subtree visible through ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - rootAlbumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.AlbumID: matching album IDs
//   - error: query, scan, or row iteration error.
func CollectAlbumSubtreeIDs(db *sql.DB, c context.Context, rootAlbumID dbo.AlbumID, acl dbo.ACLContext) ([]dbo.AlbumID, error) {
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

// CountAlbumSubtree counts child albums and images in a visible album subtree.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - rootAlbumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used to build visibility filters.
//
// Output:
//   - int: child album count.
//   - int: image count.
//   - error: album ID query or image count error.
func CountAlbumSubtree(db *sql.DB, c context.Context, rootAlbumID dbo.AlbumID, acl dbo.ACLContext) (int, int, error) {
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

// QueryAlbumGraph reads album graph nodes with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - []dbo.AlbumGraph: album graph nodes with ID, parent ID, and name.
//   - error: query, scan, or row iteration error.
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

// QueryAlbumsIdByCover reads album IDs that use an image as cover.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - cover: cover image ID.
//
// Output:
//   - []dbo.AlbumID: album IDs using the cover image.
//   - error: query, scan, or row iteration error.
func QueryAlbumsIdByCover(db *sql.DB, c context.Context, cover dbo.ImageID) ([]dbo.AlbumID, error) {
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

// CountAlbum counts all albums with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - uint64: album count.
//   - error: query or scan error, if any.
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

// QueryAlbumIDByImageID reads album IDs containing an image.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image ID to look up.
//
// Output:
//   - []dbo.AlbumID: album IDs containing the image.
//   - error: query, scan, or row iteration error.
func QueryAlbumIDByImageID(db *sql.DB, c context.Context, imageID dbo.ImageID) ([]dbo.AlbumID, error) {
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

// QueryAlbumsIDImageIDACL reads albums containing an image and visible through ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image ID to look up.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.Album: matching albums.
//   - error: query, scan, decode, or row iteration error.
func QueryAlbumsIDImageIDACL(db *sql.DB, c context.Context, imageID dbo.ImageID, acl dbo.ACLContext) ([]dbo.Album, error) {
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

// ReorderAllImages recalculates album-image positions.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - error: exec error, if any.
func ReorderAllImages(db *sql.DB, c context.Context) error {
	logScope, ctx := logging.Enter(c, "dao/album/update/image/reorder", nil, nil)
	q := NewQueries(db)
	err := q.ReorderAllImage(ctx)
	return logging.Return(logScope, err)
}

// CountAlbumDescendantIDsByACL counts descendant albums visible through ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching album count.
//   - error: query or scan error, if any.
func CountAlbumDescendantIDsByACL(db *sql.DB, c context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/albums/descendant/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumID,
	})
	q := NewQueries(db)
	qty, err := q.CountAlbumDescendantIDsByACL(ctx, albumID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// GetAlbumByIDACL reads an album by ID when visible through ACL with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to read.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.Album: matching album.
//   - error: wrapped not-found, query, scan, or decode error.
func GetAlbumByIDACL(db *sql.DB, c context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "dao/album/get/byID/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumID,
	})
	q := NewQueries(db)
	album, err := q.GetAlbumByIDACL(ctx, albumID, acl)
	return album, returnWrapNotFound(logScope, err, "album")
}
