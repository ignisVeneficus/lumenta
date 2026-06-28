package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	"database/sql"
)

const imageFields = `
i.id, i.root, i.path, i.filename, i.ext,
i.file_size, i.mtime, i.file_hash, i.meta_hash,
i.title, i.caption,
i.taken_at, i.camera, i.lens, i.focal_length, i.aperture, i.exposure, i.iso,
i.latitude, i.longitude, i.rotation, i.rating, i.width, i.height, i.panorama,
i.focus_x, i.focus_y, i.focus_mode, i.focus_source,
i.exif_json,
i.acl_level, i.acl_user_id, i.acl_source,
i.created_at, i.updated_at, i.last_seen_sync
`

const defaultImageOrderForward = ` ORDER BY i.order_date ASC, i.filename ASC, i.id ASC`
const defaultImageOrderBackward = ` ORDER BY i.order_date DESC, i.filename DESC, i.id DESC`

const imageWhereNext = ` (
      i.order_date >  ?
   OR (i.order_date = ? AND i.filename > ?)
   OR (i.order_date = ? AND i.filename = ? AND i.id > ?)
)
`
const imageWherePrev = ` (
      i.order_date <  ?
   OR (i.order_date = ? AND i.filename < ?)
   OR (i.order_date = ? AND i.filename = ? AND i.id < ?)
)
`

const getImageById = `SELECT ` + imageFields + ` FROM images i WHERE i.id=?`

const getImageByIdACL = `SELECT ` + imageFields + ` FROM images i WHERE i.id=? AND %s `

const createImage = `
INSERT INTO images (
  root, path, filename, ext,
  file_size, mtime, file_hash, meta_hash,
  title, caption,
  taken_at, order_date, 
  camera, lens, focal_length, aperture, exposure, iso,
  latitude, longitude, rotation, rating, width, height, panorama,
  focus_x, focus_y, focus_mode, focus_source,
  exif_json,
  acl_level, acl_user_id, acl_source, last_seen_sync
) VALUES (?,?,?,?,?,?,?,?,?,?,?,IFNULL(taken_at, '1000-01-01 00:00:00'),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

const deleteImage = `DELETE FROM images WHERE id=?`

const updateImage = `
UPDATE images SET
  root=?, path=?, filename=?, ext=?,
  file_size=?, mtime=?, file_hash=?, meta_hash=?,
  title=?, caption=?,
  taken_at=?, order_date = IFNULL(taken_at, '1000-01-01 00:00:00'),
  camera=?, lens=?, focal_length=?, aperture=?, exposure=?, iso=?,
  latitude=?, longitude=?, rotation=?, rating=?, width=?, height=?, panorama=?,
  focus_x=?, focus_y=?, focus_mode=?, focus_source=?, 
  exif_json=?,
  acl_level=?, acl_user_id=?, acl_source=?, last_seen_sync=?
WHERE id=?`

const bindImageTag = `INSERT IGNORE INTO image_tags (image_id, tag_id) VALUES (?,?)`
const breakImageTag = `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`
const breakImageAllTag = `DELETE FROM image_tags WHERE image_id = ?`

const getImageByPath = `
SELECT ` + imageFields + `FROM images i WHERE i.root=? AND i.path = ? AND i.filename = ? AND i.ext = ? `

const countImageByAlbums = `
SELECT COUNT(DISTINCT ai.image_id)
FROM album_images ai
JOIN images i ON i.id = ai.image_id
WHERE ai.album_id IN (%s)
AND %s `

const queryImageIDsByAlbumsPage = `
SELECT ai.image_id
FROM album_images ai
JOIN images i ON i.id = ai.image_id
WHERE ai.album_id IN (%s)
AND ai.image_id > ?
AND %s
GROUP BY ai.image_id
ORDER BY ai.image_id
LIMIT ?`

const deleteImageNotSeen = `DELETE FROM images WHERE (last_seen_sync IS NULL OR last_seen_sync <> ?)
LIMIT ?`

const updateImageSyncID = `UPDATE images SET last_seen_sync=? WHERE id=?`

const queryImageWLastSyncWUserByPathPaged = `SELECT ` + imageFields + ` , sr.finished_at, u.username FROM images AS i LEFT JOIN sync_runs AS sr ON i.last_seen_sync=sr.id LEFT JOIN users AS u ON u.id=i.acl_user_id WHERE i.root=? AND i.path = ? ORDER BY i.filename, i.ext LIMIT ?,?`
const countImagesByPath = `SELECT count(*) FROM images AS i WHERE i.root = ? AND i.path = ?`

const getImageIdByPathFirstHash = `SELECT i.id FROM images AS i WHERE i.root = ? AND (i.path = ? OR i.path like CONCAT(?,'/%')) ORDER BY i.file_hash LIMIT 1`
const getImageIdByRootFirstHash = `SELECT i.id FROM images AS i WHERE i.root = ? ORDER BY i.file_hash LIMIT 1`

const queryImagePathByParentPathInner = `
SELECT DISTINCT
  SUBSTRING_INDEX(path, '/', ? + 1) AS child_path
FROM images
WHERE
  root = ?
  AND SUBSTRING_INDEX(path, '/', ?) = ?
  AND path <> ?
`
const queryImagePathByParentPathPaged = queryImagePathByParentPathInner + ` LIMIT ?,?`
const countImagePathByParentPath = `SELECT COUNT(*) AS total FROM (` + queryImagePathByParentPathInner + `) t`

const countImageByParentPath = `SELECT count(*) FROM images AS i WHERE i.root= ? AND i.path = ? OR i.path like CONCAT(?,'/%') `
const countImageByRoot = `SELECT count(*) FROM images AS i WHERE i.root= ?`

const imageByTagACLWhere = ` FROM images AS i JOIN image_tags AS it ON i.id = it.image_id WHERE it.tag_id = ? AND %s `
const queryImageByTagACLPaged = `SELECT ` + imageFields + imageByTagACLWhere + defaultImageOrderForward + ` LIMIT ?, ? `
const countImageByTagACL = `SELECT COUNT(*) ` + imageByTagACLWhere

const getImageIdByTagACLFirstHash = `SELECT i.id ` + imageByTagACLWhere + ` ORDER BY i.file_hash LIMIT 1 `

const countImageRoots = `SELECT COUNT(DISTINCT i.root) AS root_count FROM images i`
const queryImageRoots = `SELECT DISTINCT i.root FROM images i order by i.root`

const queryImageIDByTagACLNext = `SELECT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name ` + imageByTagACLWhere + " AND " + imageWhereNext + defaultImageOrderForward + ` LIMIT ?, ?`
const queryImageIDByTagACLPrev = `SELECT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name ` + imageByTagACLWhere + " AND " + imageWherePrev + defaultImageOrderBackward + ` LIMIT ?, ?`

const queryImageCoordByTagACL = `SELECT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name, i.latitude, i.longitude` + imageByTagACLWhere + defaultImageOrderForward

const queryImageRandomByHashByACLForward = `
SELECT ` + imageFields + ` FROM images AS i WHERE i.file_hash >=? AND %s ORDER BY i.file_hash, i.id LIMIT ? `
const queryImageRandomByHashByACLBackward = `
SELECT ` + imageFields + ` FROM images AS i WHERE i.file_hash <? AND %s ORDER BY i.file_hash desc, i.id desc LIMIT ? `

const countImageByLastNotSeen = `
SELECT COUNT(*) FROM images AS i WHERE (last_seen_sync IS NULL OR last_seen_sync <> ?)`

const countImageByLastSeen = `
SELECT COUNT(*) FROM images AS i WHERE last_seen_sync = ?`

const countImage = `
SELECT COUNT(*) FROM images`

const countImageACLLevels = `
SELECT i.acl_level, COUNT(*) FROM images AS i GROUP BY i.acl_level`

const countImageByAlbumACL = `SELECT count(i.id)
FROM album_images ai
JOIN images i ON i.id = ai.image_id
AND %s 
WHERE ai.album_id=? `

const queryImageByAlbumACLPaged = `SELECT ` + imageFields + `
FROM album_images ai
JOIN images i ON i.id = ai.image_id
AND %s 
WHERE ai.album_id=? 
ORDER BY ai.position
LIMIT ?,?
`

const countImagesByAlbumDescendantIDsByACL = `SELECT count(DISTINCT i.id)
FROM albums a
JOIN album_images ai ON a.ID = ai.album_id
JOIN images i ON i.id = ai.image_id
AND %s 
WHERE JSON_CONTAINS(a.ancestor_ids, ?)
AND %s `

const queryImageCoordByAlbumDescendantIDsByACL = `SELECT DISTINCT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name, i.latitude, i.longitude
FROM albums a
JOIN album_images ai ON a.ID = ai.album_id
JOIN images i ON i.id = ai.image_id
AND %s 
WHERE JSON_CONTAINS(a.ancestor_ids, ?)
AND %s `

const queryImageCoordByAlbumRootByACL = `
SELECT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name, i.latitude, i.longitude
FROM images i
INNER JOIN album_images AS ai
ON i.id = ai.image_id
WHERE 1=1
AND %s
`

const queryImageIDByAlbumACLBegin = `
SELECT i.id,COALESCE(NULLIF(i.title,''), i.filename) AS display_name FROM album_images AS ai
JOIN images AS i
ON i.id = ai.image_id
WHERE ai.album_id = ?
AND %s
`

const queryImageIDByAlbumACLNext = queryImageIDByAlbumACLBegin +
	`
AND ai.position > (
	SELECT position FROM album_images
	WHERE album_id = ? AND image_id = ?
)
ORDER BY ai.position
LIMIT ?,?
`
const queryImageIDByAlbumACLPrev = queryImageIDByAlbumACLBegin +
	`
AND ai.position < (
	SELECT position FROM album_images
	WHERE album_id = ? AND image_id = ?
)
ORDER BY ai.position DESC
LIMIT ?,?
`

const countImageByAlbumsBinding = `
SELECT
    COALESCE(SUM(CASE WHEN ai.image_id IS NOT NULL THEN 1 ELSE 0 END), 0) AS with_album,
    COALESCE(SUM(CASE WHEN ai.image_id IS NULL THEN 1 ELSE 0 END), 0) AS without_album
FROM images i
LEFT JOIN (
    SELECT DISTINCT image_id
    FROM album_images
) ai ON ai.image_id = i.id;
`

const patchImage = `
UPDATE images SET
  focus_x=?, focus_y=?, focus_mode=?, focus_source=?,
  acl_level=?, acl_user_id=?, acl_source=?
WHERE id=?`

// parseImage scans a single image row into an Image value.
//
// Input:
//   - row: SQL row containing the imageFields columns.
//
// Output:
//   - dbo.Image: scanned image.
//   - error: scan error, if any.
func parseImage(row *sql.Row) (dbo.Image, error) {
	var i dbo.Image
	err := row.Scan(
		&i.ID,
		&i.Root,
		&i.Path,
		&i.Filename,
		&i.Ext,
		&i.FileSize,
		&i.MTime,
		&i.FileHash,
		&i.MetaHash,
		&i.Title,
		&i.Caption,
		&i.TakenAt,
		&i.Camera,
		&i.Lens,
		&i.FocalLength,
		&i.Aperture,
		&i.Exposure,
		&i.ISO,
		&i.Latitude,
		&i.Longitude,
		&i.Rotation,
		&i.Rating,
		&i.Width,
		&i.Height,
		&i.Panorama,
		&i.FocusX,
		&i.FocusY,
		&i.FocusMode,
		&i.FocusSource,
		&i.ExifJSON,
		&i.ACLLevel,
		&i.ACLUserID,
		&i.ACLSource,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.LastSeenSync,
	)
	return i, err
}

// getImageParamsNextPrev builds ordering parameters for next/previous image queries.
//
// Input:
//   - imageID: image ID used as a tie-breaker in ordering.
//   - orderDate: order date used for comparison; nil uses the database default date.
//   - filename: filename used for comparison.
//
// Output:
//   - []any: SQL parameters for next/previous ordering predicates.
func getImageParamsNextPrev(imageID dbo.ImageID, orderDate *time.Time, filename string) []any {
	if orderDate == nil {
		// default from database
		oD, _ := time.Parse("2006-01-02 15:04:05", "1000-01-01 00:00:00")
		orderDate = &oD
	}
	return []any{
		orderDate, orderDate, filename, orderDate, filename, imageID,
	}
}

// parseImageRows scans image rows into Image values.
//
// Input:
//   - rows: SQL rows containing the imageFields columns.
//
// Output:
//   - []dbo.Image: scanned images.
//   - error: scan or row iteration error.
func parseImageRows(rows *sql.Rows) ([]dbo.Image, error) {
	out := make([]dbo.Image, 0)
	for rows.Next() {
		var i dbo.Image
		err := rows.Scan(
			&i.ID,
			&i.Root,
			&i.Path,
			&i.Filename,
			&i.Ext,
			&i.FileSize,
			&i.MTime,
			&i.FileHash,
			&i.MetaHash,
			&i.Title,
			&i.Caption,
			&i.TakenAt,
			&i.Camera,
			&i.Lens,
			&i.FocalLength,
			&i.Aperture,
			&i.Exposure,
			&i.ISO,
			&i.Latitude,
			&i.Longitude,
			&i.Rotation,
			&i.Rating,
			&i.Width,
			&i.Height,
			&i.Panorama,
			&i.FocusX,
			&i.FocusY,
			&i.FocusMode,
			&i.FocusSource,
			&i.ExifJSON,
			&i.ACLLevel,
			&i.ACLUserID,
			&i.ACLSource,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.LastSeenSync,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetImageById reads an image by ID.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to read.
//
// Output:
//   - dbo.Image: matching image.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageById(ctx context.Context, imageID dbo.ImageID) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageById, imageID)
	return parseImage(row)
}

// GetImageByIdACL reads an image by ID when visible through ACL.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to read.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.Image: matching image.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageByIdACL(ctx context.Context, imageID dbo.ImageID, acl dbo.ACLContext) (dbo.Image, error) {
	sql, params := CreateAclWhere("i", acl)
	params = append([]any{imageID}, params...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(getImageByIdACL, sql), params...)
	return parseImage(row)
}

// CreateImage inserts an image.
//
// Input:
//   - ctx: request context.
//   - i: image data to insert.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) CreateImage(ctx context.Context, i dbo.Image) error {
	_, err := q.db.ExecContext(
		ctx,
		createImage,
		i.Root,
		i.Path,
		i.Filename,
		i.Ext,
		i.FileSize,
		i.MTime,
		i.FileHash,
		i.MetaHash,
		i.Title,
		i.Caption,
		i.TakenAt,
		i.Camera,
		i.Lens,
		i.FocalLength,
		i.Aperture,
		i.Exposure,
		i.ISO,
		i.Latitude,
		i.Longitude,
		i.Rotation,
		i.Rating,
		i.Width,
		i.Height,
		i.Panorama,
		i.FocusX,
		i.FocusY,
		i.FocusMode,
		i.FocusSource,
		i.ExifJSON,
		i.ACLLevel,
		i.ACLUserID,
		i.ACLSource,
		i.LastSeenSync,
	)
	return err
}

// DeleteImage deletes an image by ID.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to delete.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) DeleteImage(ctx context.Context, imageID dbo.ImageID) error {
	_, err := q.db.ExecContext(ctx, deleteImage, imageID)
	return err
}

// UpdateImage updates an image row.
//
// Input:
//   - ctx: request context.
//   - i: image data to write; i.ID identifies the row.
//
// Output:
//   - error: sql.ErrNoRows for nil ID, or exec error.
func (q *Queries) UpdateImage(ctx context.Context, i dbo.Image) error {
	if i.ID == nil {
		return sql.ErrNoRows
	}
	_, err := q.db.ExecContext(
		ctx,
		updateImage,
		i.Root,
		i.Path,
		i.Filename,
		i.Ext,
		i.FileSize,
		i.MTime,
		i.FileHash,
		i.MetaHash,
		i.Title,
		i.Caption,
		i.TakenAt,
		i.Camera,
		i.Lens,
		i.FocalLength,
		i.Aperture,
		i.Exposure,
		i.ISO,
		i.Latitude,
		i.Longitude,
		i.Rotation,
		i.Rating,
		i.Width,
		i.Height,
		i.Panorama,
		i.FocusX,
		i.FocusY,
		i.FocusMode,
		i.FocusSource,
		i.ExifJSON,
		i.ACLLevel,
		i.ACLUserID,
		i.ACLSource,
		i.LastSeenSync,
		i.ID,
	)
	return err
}

// BindImageTag creates an image-tag relation.
//
// Input:
//   - ctx: request context.
//   - imageID: image to bind.
//   - tagID: tag to bind.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) BindImageTag(ctx context.Context, imageID dbo.ImageID, tagID dbo.TagID) error {
	_, err := q.db.ExecContext(ctx, bindImageTag, imageID, tagID)
	return err
}

// BreakImageTag removes an image-tag relation.
//
// Input:
//   - ctx: request context.
//   - imageID: image in the relation.
//   - tagID: tag in the relation.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) BreakImageTag(ctx context.Context, imageID dbo.ImageID, tagID dbo.TagID) error {
	_, err := q.db.ExecContext(ctx, breakImageTag, imageID, tagID)
	return err
}

// BreakImageAllTag removes all tag relations for an image.
//
// Input:
//   - ctx: request context.
//   - imageID: image whose tags are removed.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) BreakImageAllTag(ctx context.Context, imageID dbo.ImageID) error {
	_, err := q.db.ExecContext(ctx, breakImageAllTag, imageID)
	return err
}

// GetImageByPath reads an image by filesystem identity.
//
// Input:
//   - ctx: request context.
//   - root, path, filename, ext: filesystem identity fields.
//
// Output:
//   - dbo.Image: matching image.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageByPath(ctx context.Context, root, path, filename, ext string) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageByPath, root, path, filename, ext)
	return parseImage(row)
}

// CountImageByAlbums counts distinct visible images in the supplied albums.
//
// Input:
//   - ctx: request context.
//   - albumIDs: album IDs to search.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: distinct visible image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByAlbums(ctx context.Context, albumIDs []dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	if len(albumIDs) == 0 {
		return 0, nil
	}
	aclWhere, aclParams := CreateAclWhere("i", acl)

	inClause, args := buildUint64InClause(albumIDs)

	query := fmt.Sprintf(countImageByAlbums, inClause, aclWhere)
	params := append(args, aclParams...)

	row := q.db.QueryRowContext(ctx, query, params...)

	var cnt uint64
	err := row.Scan(&cnt)
	return cnt, err
}

// QueryImageIDsByAlbumsPage reads visible image IDs from albums after an image ID cursor.
//
// Input:
//   - ctx: request context.
//   - albumIDs: album IDs to search.
//   - acl: ACL context used to build the visibility filter.
//   - from: image ID cursor.
//   - qty: maximum number of image IDs.
//
// Output:
//   - []dbo.ImageID: matching image IDs.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageIDsByAlbumsPage(ctx context.Context, albumIDs []dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageID, error) {
	if len(albumIDs) == 0 || qty == 0 {
		return []dbo.ImageID{}, nil
	}
	aclWhere, aclParams := CreateAclWhere("i", acl)

	inClause, args := buildUint64InClause(albumIDs)
	query := fmt.Sprintf(queryImageIDsByAlbumsPage, inClause, aclWhere)

	params := make([]any, 0, len(args)+len(aclParams)+2)
	params = append(params, args...)
	params = append(params, from)
	params = append(params, aclParams...)
	params = append(params, qty)

	rows, err := q.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]dbo.ImageID, 0, qty)
	for rows.Next() {
		var id dbo.ImageID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteImageNotSeen deletes images not seen in the given sync run.
//
// Input:
//   - ctx: request context.
//   - syncID: sync run used as the current seen marker.
//   - limit: maximum number of rows to delete.
//
// Output:
//   - uint64: number of deleted rows.
//   - error: exec or row-count error, if any.
func (q *Queries) DeleteImageNotSeen(ctx context.Context, syncID dbo.SyncRunID, limit uint32) (uint64, error) {
	res, err := q.db.ExecContext(ctx, deleteImageNotSeen, syncID, limit)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint64(affected), nil
}

// UpdateImageSyncID updates the last seen sync marker for an image.
//
// Input:
//   - ctx: request context.
//   - imageID: image ID to update.
//   - syncID: sync run to store as last_seen_sync.
//
// Output:
//   - error: exec error, if any.
func (q *Queries) UpdateImageSyncID(ctx context.Context, imageID dbo.ImageID, syncID dbo.SyncRunID) error {
	_, err := q.db.ExecContext(ctx, updateImageSyncID, syncID, imageID)
	return err
}

// QueryImageWLastSyncByWUserPathPaged reads images in a path with last sync and user data.
//
// Input:
//   - ctx: request context.
//   - root, path: filesystem location to query.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageWLastSyncWUser: matching images with joined sync and user fields.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageWLastSyncByWUserPathPaged(ctx context.Context, root, path string, from, qty uint64) ([]dbo.ImageWLastSyncWUser, error) {
	rows, err := q.db.QueryContext(ctx, queryImageWLastSyncWUserByPathPaged, root, path, from, qty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageWLastSyncWUser, 0)
	for rows.Next() {
		var i dbo.ImageWLastSyncWUser
		err := rows.Scan(
			&i.ID,
			&i.Root,
			&i.Path,
			&i.Filename,
			&i.Ext,
			&i.FileSize,
			&i.MTime,
			&i.FileHash,
			&i.MetaHash,
			&i.Title,
			&i.Caption,
			&i.TakenAt,
			&i.Camera,
			&i.Lens,
			&i.FocalLength,
			&i.Aperture,
			&i.Exposure,
			&i.ISO,
			&i.Latitude,
			&i.Longitude,
			&i.Rotation,
			&i.Rating,
			&i.Width,
			&i.Height,
			&i.Panorama,
			&i.FocusX,
			&i.FocusY,
			&i.FocusMode,
			&i.FocusSource,
			&i.ExifJSON,
			&i.ACLLevel,
			&i.ACLUserID,
			&i.ACLSource,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.LastSeenSync,
			&i.LastSyncDate,
			&i.User,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// CountImageByPath counts images in a filesystem path.
//
// Input:
//   - ctx: request context.
//   - root, path: filesystem location to count.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByPath(ctx context.Context, root, path string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImagesByPath, root, path)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// GetImageIdByPathFirstHash reads the first image ID under a path ordered by file hash.
//
// Input:
//   - ctx: request context.
//   - root, path: filesystem root and path prefix.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageIdByPathFirstHash(ctx context.Context, root, path string) (dbo.ImageID, error) {
	row := q.db.QueryRowContext(ctx, getImageIdByPathFirstHash, root, path, path)
	var id dbo.ImageID
	err := row.Scan(&id)
	return id, err
}

// GetImageIdByRootFirstHash reads the first image ID under a root ordered by file hash.
//
// Input:
//   - ctx: request context.
//   - root: filesystem root to search.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageIdByRootFirstHash(ctx context.Context, root string) (dbo.ImageID, error) {
	row := q.db.QueryRowContext(ctx, getImageIdByRootFirstHash, root)
	var id dbo.ImageID
	err := row.Scan(&id)
	return id, err
}

// QueryImagePathByParentPathPaged reads distinct child paths under a parent path.
//
// Input:
//   - ctx: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path whose children are returned.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []string: distinct child paths.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImagePathByParentPathPaged(ctx context.Context, root, parentPath string, from, qty uint64) ([]string, error) {
	depth := 0
	if parentPath != "" {
		depth = strings.Count(parentPath, "/") + 1
	}
	rows, err := q.db.QueryContext(ctx, queryImagePathByParentPathPaged, depth, root, depth, parentPath, parentPath, from, qty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 8)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CountImagePathByParentPath counts distinct child paths under a parent path.
//
// Input:
//   - ctx: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path whose children are counted.
//
// Output:
//   - uint64: distinct child path count.
//   - error: query or scan error, if any.
func (q *Queries) CountImagePathByParentPath(ctx context.Context, root, parentPath string) (uint64, error) {
	depth := 0
	if parentPath != "" {
		depth = strings.Count(parentPath, "/") + 1
	}
	row := q.db.QueryRowContext(ctx, countImagePathByParentPath, depth, root, depth, parentPath, parentPath)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImageByParentPath counts images under a parent path.
//
// Input:
//   - ctx: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path or path prefix.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByParentPath(ctx context.Context, root, parentPath string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByParentPath, root, parentPath, parentPath)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImageByRoot counts images under a filesystem root.
//
// Input:
//   - ctx: request context.
//   - root: filesystem root to count.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByRoot(ctx context.Context, root string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByRoot, root)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// QueryImageByTagACLPaged reads images bound to a tag and visible through ACL.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageByTagACLPaged(ctx context.Context, tagID dbo.TagID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagID}
	params = append(params, aclParams...)
	params = append(params, from)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageByTagACLPaged, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images, err := parseImageRows(rows)
	return images, err
}

// CountImageByTagACL counts images bound to a tag and visible through ACL.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByTagACL(ctx context.Context, tagID dbo.TagID, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagID}
	params = append(params, aclParams...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(countImageByTagACL, aclWhere), params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// GetImageIdByTagACLFirstHash reads the first visible image ID for a tag ordered by file hash.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: query, scan, or sql.ErrNoRows error.
func (q *Queries) GetImageIdByTagACLFirstHash(ctx context.Context, tagID dbo.TagID, acl dbo.ACLContext) (dbo.ImageID, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagID}
	params = append(params, aclParams...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(getImageIdByTagACLFirstHash, aclWhere), params...)
	var id dbo.ImageID
	err := row.Scan(&id)
	return id, err
}

// CountImageRoots counts distinct image roots.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - uint64: distinct root count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageRoots(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageRoots)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// QueryImageRoots reads distinct image roots.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - []string: distinct root names.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageRoots(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, queryImageRoots)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 8)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageIDByTagACLNext reads next image IDs and titles for a tag under ACL.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - imageID: current image ID used as an ordering cursor.
//   - orderDate: current image order date; nil uses the database default date.
//   - fileName: current image filename used as an ordering cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: next image IDs with display titles.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageIDByTagACLNext(ctx context.Context, tagID dbo.TagID, imageID dbo.ImageID, orderDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagID}
	params = append(params, aclParams...)
	params = append(params, getImageParamsNextPrev(imageID, orderDate, fileName)...)
	params = append(params, from, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageIDByTagACLNext, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageTitle, 0)
	for rows.Next() {
		var i dbo.ImageTitle
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageIDByTagACLPrev reads previous image IDs and titles for a tag under ACL.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - imageID: current image ID used as an ordering cursor.
//   - orderDate: current image order date; nil uses the database default date.
//   - fileName: current image filename used as an ordering cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: previous image IDs with display titles.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageIDByTagACLPrev(ctx context.Context, tagID dbo.TagID, imageID dbo.ImageID, orderDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagID}
	params = append(params, aclParams...)
	params = append(params, getImageParamsNextPrev(imageID, orderDate, fileName)...)
	params = append(params, from, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageIDByTagACLPrev, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageTitle, 0)
	for rows.Next() {
		var i dbo.ImageTitle
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil

}

// QueryImageCoordByTagACL reads visible image coordinates for a tag.
//
// Input:
//   - ctx: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.ImageCoord: image IDs, titles, and coordinates.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageCoordByTagACL(ctx context.Context, tagID dbo.TagID, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagID}
	params = append(params, aclParams...)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageCoordByTagACL, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageCoord, 0)
	for rows.Next() {
		var i dbo.ImageCoord
		if err := rows.Scan(&i.ID, &i.Title, &i.Latitude, &i.Longitude); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageRandomByHashByACLForward reads visible images at or after a hash value.
//
// Input:
//   - ctx: request context.
//   - hash: file hash lower bound.
//   - acl: ACL context used to build the visibility filter.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageRandomByHashByACLForward(ctx context.Context, hash string, acl dbo.ACLContext, qty int) ([]dbo.Image, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{hash}
	params = append(params, aclParams...)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageRandomByHashByACLForward, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images, err := parseImageRows(rows)
	return images, err
}

// QueryImageRandomByHashByACLBackward reads visible images before a hash value.
//
// Input:
//   - ctx: request context.
//   - hash: file hash upper bound.
//   - acl: ACL context used to build the visibility filter.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageRandomByHashByACLBackward(ctx context.Context, hash string, acl dbo.ACLContext, qty int) ([]dbo.Image, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{hash}
	params = append(params, aclParams...)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageRandomByHashByACLBackward, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images, err := parseImageRows(rows)
	return images, err
}

// CountImageByLastNotSeen counts images not seen in a sync run.
//
// Input:
//   - ctx: request context.
//   - lastSyncID: sync run used as the current seen marker.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByLastNotSeen(ctx context.Context, lastSyncID dbo.SyncRunID) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByLastNotSeen, lastSyncID)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImageByLastSeen counts images seen in a sync run.
//
// Input:
//   - ctx: request context.
//   - lastSyncID: sync run used as the seen marker.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByLastSeen(ctx context.Context, lastSyncID dbo.SyncRunID) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByLastSeen, lastSyncID)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImage counts all images.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - uint64: image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImage(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImage)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImageACLLevels counts images grouped by ACL level.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - dbo.ImageACLCount: counts per ACL level.
//   - error: query, scan, or row iteration error.
func (q *Queries) CountImageACLLevels(ctx context.Context) (dbo.ImageACLCount, error) {
	rows, err := q.db.QueryContext(ctx, countImageACLLevels)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := dbo.ImageACLCount{}
	for rows.Next() {
		var (
			acl_level uint64
			count     uint64
		)
		if err := rows.Scan(&acl_level, &count); err != nil {
			return nil, err
		}
		out[dbo.DBACLLevel(acl_level)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageByAlbumACLPaged reads images in an album and visible through ACL.
//
// Input:
//   - ctx: request context.
//   - albumId: album ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageByAlbumACLPaged(ctx context.Context, albumId dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclParams...)
	params = append(params, albumId)
	params = append(params, from)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageByAlbumACLPaged, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images, err := parseImageRows(rows)
	return images, err
}

// CountImageByAlbumACL counts images in an album and visible through ACL.
//
// Input:
//   - ctx: request context.
//   - albumId: album ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByAlbumACL(ctx context.Context, albumId dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclParams...)
	params = append(params, albumId)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(countImageByAlbumACL, aclWhere), params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

// CountImageByAlbumDescendantIDsByACL counts distinct images in descendant albums visible through ACL.
//
// Input:
//   - ctx: request context.
//   - albumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used for album and image visibility filters.
//
// Output:
//   - uint64: distinct visible image count.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByAlbumDescendantIDsByACL(ctx context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	aclAWhere, aclAParams := CreateAclWhere("a", acl)
	aclIWhere, aclIParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclIParams...)
	params = append(params, fmt.Sprintf("[%d]", albumID))
	params = append(params, aclAParams...)
	sql := fmt.Sprintf(countImagesByAlbumDescendantIDsByACL, aclIWhere, aclAWhere)
	row := q.db.QueryRowContext(ctx, sql, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) QueryImageCoordByAlbumDescendantIDsByACL(ctx context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	aclAWhere, aclAParams := CreateAclWhere("a", acl)
	aclIWhere, aclIParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclIParams...)
	params = append(params, fmt.Sprintf("[%d]", albumID))
	params = append(params, aclAParams...)
	sql := fmt.Sprintf(queryImageCoordByAlbumDescendantIDsByACL, aclIWhere, aclAWhere)
	rows, err := q.db.QueryContext(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageCoord, 0)
	for rows.Next() {
		var i dbo.ImageCoord
		if err := rows.Scan(&i.ID, &i.Title, &i.Latitude, &i.Longitude); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queries) QueryImageCoordByAlbumRootByACL(ctx context.Context, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	sql := fmt.Sprintf(queryImageCoordByAlbumRootByACL, aclWhere)
	rows, err := q.db.QueryContext(ctx, sql, aclParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageCoord, 0)
	for rows.Next() {
		var i dbo.ImageCoord
		if err := rows.Scan(&i.ID, &i.Title, &i.Latitude, &i.Longitude); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageIDByAlbumACLNext reads next image IDs and titles in an album under ACL.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to filter by.
//   - imageID: current image ID used as a position cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: next image IDs with display titles.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageIDByAlbumACLNext(ctx context.Context, albumID dbo.AlbumID, imageID dbo.ImageID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{albumID}
	params = append(params, aclParams...)
	params = append(params, albumID, imageID, from, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageIDByAlbumACLNext, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageTitle, 0)
	for rows.Next() {
		var i dbo.ImageTitle
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// QueryImageIDByAlbumACLPrev reads previous image IDs and titles in an album under ACL.
//
// Input:
//   - ctx: request context.
//   - albumID: album ID to filter by.
//   - imageID: current image ID used as a position cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: previous image IDs with display titles.
//   - error: query, scan, or row iteration error.
func (q *Queries) QueryImageIDByAlbumACLPrev(ctx context.Context, albumID dbo.AlbumID, imageID dbo.ImageID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{albumID}
	params = append(params, aclParams...)
	params = append(params, albumID, imageID, from, qty)
	rows, err := q.db.QueryContext(ctx, fmt.Sprintf(queryImageIDByAlbumACLPrev, aclWhere), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]dbo.ImageTitle, 0)
	for rows.Next() {
		var i dbo.ImageTitle
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CountImageByAlbumsBinding counts images with and without album bindings.
//
// Input:
//   - ctx: request context.
//
// Output:
//   - uint64: number of images with at least one album binding.
//   - uint64: number of images without album binding.
//   - error: query or scan error, if any.
func (q *Queries) CountImageByAlbumsBinding(ctx context.Context) (uint64, uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByAlbumsBinding)
	var withAlbum uint64
	var withoutAlbum uint64
	err := row.Scan(&withAlbum, &withoutAlbum)
	return withAlbum, withoutAlbum, err
}

func (q *Queries) PatchImage(ctx context.Context, i dbo.Image) error {
	if i.ID == nil {
		return sql.ErrNoRows
	}
	_, err := q.db.ExecContext(
		ctx,
		patchImage,
		i.FocusX,
		i.FocusY,
		i.FocusMode,
		i.FocusSource,
		i.ACLLevel,
		i.ACLUserID,
		i.ACLSource,
		i.ID,
	)
	return err
}

//
// =========================================================
// Public API functions
// =========================================================
//

// BindImageTag creates an image-tag relation in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image to bind.
//   - tagID: tag to bind.
//
// Output:
//   - error: transaction, bind, or commit error.
func BindImageTag(db *sql.DB, c context.Context, imageID dbo.ImageID, tagID dbo.TagID) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/bind", tagID, map[string]any{
		"image_id": imageID,
		"tag_id":   tagID,
	})
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindImageTag(ctx, imageID, tagID); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

// CreateImage inserts an image and writes the new ID back to i.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - i: image to insert; receives the new ID.
//
// Output:
//   - dbo.ImageID: created image ID.
//   - error: transaction, insert, ID lookup, tag bind, or commit error.
func CreateImage(db *sql.DB, c context.Context, i *dbo.Image) (dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/create", i.Root+"/"+i.Path+"/"+i.Filename+"."+i.Ext, map[string]any{
		"image": i,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateImage(ctx, *i); err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}

	id, err := q.GetLastId(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	newID := dbo.ImageID(id)
	i.ID = &newID

	return newID, logging.ReturnParams(logScope, tx.Commit(), map[string]any{"new_id": id})
}

// UpdateImage updates an image and rewrites its tag relations.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - i: image data to update; i.ID identifies the row.
//
// Output:
//   - error: transaction, update, tag rewrite, or commit error.
func UpdateImage(db *sql.DB, c context.Context, i dbo.Image) error {
	logScope, ctx := logging.Enter(c, "dao/image/update", i.ID, map[string]any{"image": i})
	if i.ID == nil {
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
	if err := q.UpdateImage(ctx, i); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

// CreateOrUpdateImage inserts or updates an image and rewrites its tag relations.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - i: image to create or update; receives a new ID on insert.
//
// Output:
//   - dbo.ImageID: image ID after create or update.
//   - error: transaction, insert, update, ID lookup, tag rewrite, or commit error.
func CreateOrUpdateImage(db *sql.DB, c context.Context, i *dbo.Image) (dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/createOrUpdate", i.Root+"/"+i.Path+"/"+i.Filename+"."+i.Ext, map[string]any{
		"image": i,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if i.ID == nil {
		if err := q.CreateImage(ctx, *i); err != nil {
			logging.ExitErr(logScope, err)
			return 0, err
		}
		id, err := q.GetLastId(ctx)
		if err != nil {
			logging.ExitErr(logScope, err)
			return 0, err
		}
		newID := dbo.ImageID(id)
		i.ID = &newID
	} else {
		if err := q.UpdateImage(ctx, *i); err != nil {
			logging.ExitErr(logScope, err)
			return *i.ID, err
		}
	}
	q.BreakImageAllTag(ctx, *i.ID)
	for _, tag := range i.Tags {
		if err := writeTagTree(q, ctx, *tag, *i.ID); err != nil {
			logging.ExitErr(logScope, err)
			return *i.ID, err
		}
	}
	return *i.ID, logging.Return(logScope, tx.Commit())
}

// DeleteImage deletes an image in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image ID to delete.
//
// Output:
//   - error: transaction, delete, or commit error.
func DeleteImage(db *sql.DB, c context.Context, imageID dbo.ImageID) error {
	logScope, ctx := logging.Enter(c, "dao/image/delete", imageID, map[string]any{
		"image_id": imageID,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteImage(ctx, imageID); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

// GetImageByID reads an image by ID with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image ID to read.
//
// Output:
//   - dbo.Image: matching image.
//   - error: wrapped not-found, query, or scan error.
func GetImageByID(db *sql.DB, c context.Context, imageID dbo.ImageID) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byId", imageID, map[string]any{
		"image_id": imageID,
	})

	q := NewQueries(db)
	i, err := q.GetImageById(ctx, imageID)
	return i, returnWrapNotFound(logScope, err, "image")
}

// GetImageByIdACL reads an image by ID when visible through ACL with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - id: image ID to read.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.Image: matching image.
//   - error: wrapped not-found, query, or scan error.
func GetImageByIdACL(db *sql.DB, c context.Context, id dbo.ImageID, acl dbo.ACLContext) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byId/ACL", id, map[string]any{
		"image_id":    id,
		"acl_context": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	return i, returnWrapNotFound(logScope, err, "image")
}

// GetImageByIdWTags reads an image by ID and attaches its tags.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - id: image ID to read.
//
// Output:
//   - dbo.Image: matching image with tags.
//   - error: wrapped not-found, image query, or tag query error.
func GetImageByIdWTags(db *sql.DB, c context.Context, id dbo.ImageID) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/tags/byId", id, nil)
	q := NewQueries(db)
	i, err := q.GetImageById(ctx, id)
	err = wrapNotFound(err, "image")
	if err != nil {
		logging.ExitErr(logScope, err)
		return i, err
	}
	tags, err := q.QueryTagsByImageID(ctx, *i.ID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return i, err
	}
	i.Tags = dbo.TagsToPointer(tags)
	logging.Exit(logScope, "Ok", nil)
	return i, nil
}

// GetImageByIdACLWTags reads a visible image by ID and attaches its tags.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - id: image ID to read.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.Image: matching image with tags.
//   - error: wrapped not-found, image query, or tag query error.
func GetImageByIdACLWTags(db *sql.DB, c context.Context, id dbo.ImageID, acl dbo.ACLContext) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/tags/byId/ACL", id, map[string]any{
		"image_id": id,
		"acl":      acl,
	})

	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	err = wrapNotFound(err, "image")
	if err != nil {
		logging.ExitErr(logScope, err)
		return i, err
	}
	tags, err := q.QueryTagsByImageID(ctx, *i.ID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return i, err
	}
	i.Tags = dbo.TagsToPointer(tags)
	logging.Exit(logScope, "Ok", nil)
	return i, nil
}

// GetImageByPath reads an image by filesystem identity with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path, filename, ext: filesystem identity fields.
//
// Output:
//   - dbo.Image: matching image.
//   - error: wrapped not-found, query, or scan error.
func GetImageByPath(db *sql.DB, c context.Context, root, path, filename, ext string) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byPath", root+"/"+path+"/"+filename+"."+ext, map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
		"ext":      ext,
	})
	q := NewQueries(db)
	i, err := q.GetImageByPath(ctx, root, path, filename, ext)
	return i, returnWrapNotFound(logScope, err, "image")
}

// CountImageByAlbums counts distinct visible images in the supplied albums with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumIDs: album IDs to search.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: distinct visible image count.
//   - error: query or scan error, if any.
func CountImageByAlbums(db *sql.DB, c context.Context, albumIDs []dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byAlbums/ACL", nil, map[string]any{
		"acl":    acl,
		"albums": albumIDs,
	})
	q := NewQueries(db)
	cnt, err := q.CountImageByAlbums(ctx, albumIDs, acl)
	return cnt, logging.Return(logScope, err)
}

// QueryImageIDsByAlbumsPage reads visible image IDs from albums after an image ID cursor.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumIDs: album IDs to search.
//   - acl: ACL context used to build the visibility filter.
//   - from: image ID cursor.
//   - qty: maximum number of image IDs.
//
// Output:
//   - []dbo.ImageID: matching image IDs.
//   - error: query, scan, or row iteration error.
func QueryImageIDsByAlbumsPage(db *sql.DB, c context.Context, albumIDs []dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/id/byAlbums/Paged", nil, map[string]any{
		"acl":    acl,
		"albums": albumIDs,
		"from":   from,
		"qty":    qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageIDsByAlbumsPage(ctx, albumIDs, acl, from, qty)
	return images, logging.Return(logScope, err)
}

// BindImageTags creates multiple image-tag relations in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image to bind.
//   - tagIDs: tags to bind.
//
// Output:
//   - error: transaction, bind, or commit error.
func BindImageTags(db *sql.DB, c context.Context, imageID dbo.ImageID, tagIDs []dbo.TagID) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/tag/bind/multiple", imageID, map[string]any{
		"image": imageID,
		"tags":  tagIDs,
	})
	if len(tagIDs) == 0 {
		logging.Exit(logScope, "ok", nil)
		return nil
	}
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	for _, tagID := range tagIDs {
		if err := q.BindImageTag(ctx, imageID, tagID); err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
	}

	return logging.Return(logScope, tx.Commit())
}

// writeTagTree ensures a tag tree exists and binds its tags to an image.
//
// Input:
//   - q: query runner.
//   - c: request context.
//   - rootTag: root tag of the tree to write.
//   - imageId: image ID to bind tags to.
//
// Output:
//   - error: tag lookup, tag create, bind, or traversal error.
func writeTagTree(q *Queries, c context.Context, rootTag dbo.Tag, imageId dbo.ImageID) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/tag/tagTree", imageId, map[string]any{"root": rootTag})
	stack := []*dbo.Tag{&rootTag}
	for len(stack) > 0 {
		tag := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		var id dbo.TagID
		if tag.ID == nil {
			t, err := q.GetTagByParentAndName(ctx, tag.ParentID, tag.Name)
			err = wrapNotFound(err, "tag")
			switch {
			case err == nil:
				id = *t.ID
			case !errors.Is(err, ErrDataNotFound):
				logging.ExitErrParams(logScope, err, map[string]any{"reason": "sql-error"})
				return err
			default:
				id, err = CreateTagTx(q, ctx, *tag)
				if err != nil {
					logging.ExitErr(logScope, err)
					return err
				}
			}
		}
		if err := q.BindImageTag(ctx, imageId, id); err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		for i := len(tag.Children) - 1; i >= 0; i-- {
			c := tag.Children[i]
			if c.ParentID == nil {
				c.ParentID = &id
			}
			stack = append(stack, c)
		}

	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

// DeleteImageNotSeen deletes one batch of images not seen in a sync run.
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
func DeleteImageNotSeen(db *sql.DB, c context.Context, syncID dbo.SyncRunID, limit uint32) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/delete/notSeen", syncID, map[string]any{"sync_id": syncID})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	deleted, err := q.DeleteImageNotSeen(ctx, syncID, limit)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	return deleted, logging.ReturnParams(logScope, tx.Commit(), map[string]any{"deleted": deleted})
}

// DeleteImageNotSeenAll deletes images not seen in a sync run until none remain.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - syncID: sync run used as the current seen marker.
//   - limit: batch size for each delete pass.
//
// Output:
//   - error: delete error, if any.
func DeleteImageNotSeenAll(db *sql.DB, c context.Context, syncID dbo.SyncRunID, limit uint32) error {
	logScope, ctx := logging.Enter(c, "dao/image/delete/notSeen/all", syncID, map[string]any{"sync_id": syncID})
	deleted, err := DeleteImageNotSeen(db, ctx, syncID, limit)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	batch := 1
	for deleted > 0 {
		logging.Debug(logScope, "loop", map[string]any{
			"batch": batch,
		})
		deleted, err = DeleteImageNotSeen(db, ctx, syncID, limit)
		if err != nil {
			logging.ExitErr(logScope, err)
			return err
		}
		batch++
	}
	logging.Exit(logScope, "ok", map[string]any{"batch": batch})
	return nil
}

// UpdateImageSyncID updates the last seen sync marker for an image in a transaction.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - imageID: image ID to update.
//   - syncID: sync run to store as last_seen_sync.
//
// Output:
//   - error: transaction, update, or commit error.
func UpdateImageSyncID(db *sql.DB, c context.Context, imageID dbo.ImageID, syncID dbo.SyncRunID) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/syncID", imageID, map[string]any{
		"image_id": imageID,
		"sync_id":  syncID})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	err = q.UpdateImageSyncID(ctx, imageID, syncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

// QueryImageWLastSyncWUserByPathPaged reads images in a path with last sync and user data.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path: filesystem location to query.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageWLastSyncWUser: matching images with joined sync and user fields.
//   - error: query, scan, or row iteration error.
func QueryImageWLastSyncWUserByPathPaged(db *sql.DB, c context.Context, root, path string, from, qty uint64) ([]dbo.ImageWLastSyncWUser, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/wLastSync/wUser/byPath/Paged", root+"/"+path+"/", map[string]any{
		"root": root,
		"path": path,
		"from": from,
		"qty":  qty,
	})

	q := NewQueries(db)

	images, err := q.QueryImageWLastSyncByWUserPathPaged(ctx, root, path, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}

	logging.Exit(logScope, "ok", map[string]any{
		"found": len(images),
	})

	return images, nil
}

// CountImagesByPath counts images in a filesystem path with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path: filesystem location to count.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImagesByPath(db *sql.DB, c context.Context, root, path string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byPath", root+"/"+path, map[string]any{
		"path": path,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByPath(ctx, root, path)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// GetImageIdByPathFirstHash reads the first image ID under a path ordered by file hash.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root, path: filesystem root and path prefix.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: wrapped not-found, query, or scan error.
func GetImageIdByPathFirstHash(db *sql.DB, c context.Context, root, path string) (dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/id/byPath/firstHash", root+"/"+path, map[string]any{
		"root": root,
		"path": path,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByPathFirstHash(ctx, root, path)
	return i, returnWrapNotFound(logScope, err, "image")
}

// GetImageIdByRootFirstHash reads the first image ID under a root ordered by file hash.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root: filesystem root to search.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: wrapped not-found, query, or scan error.
func GetImageIdByRootFirstHash(db *sql.DB, c context.Context, root string) (dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/id/byRoot/firstHash", root, map[string]any{
		"root": root,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByRootFirstHash(ctx, root)
	return i, returnWrapNotFound(logScope, err, "image")
}

// QueryImagePathByParentPathPaged reads distinct child paths under a parent path.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path whose children are returned.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []string: distinct child paths.
//   - error: query, scan, or row iteration error.
func QueryImagePathByParentPathPaged(db *sql.DB, c context.Context, root, parentPath string, from, qty uint64) ([]string, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/path/byParentPath/paged", root+"/"+parentPath, map[string]any{
		"parentPpath": parentPath,
		"from":        from,
		"qty":         qty,
	})
	q := NewQueries(db)
	paths, err := q.QueryImagePathByParentPathPaged(ctx, root, parentPath, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(paths),
	})
	return paths, nil
}

// CountImagePathByParentPath counts distinct child paths under a parent path.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path whose children are counted.
//
// Output:
//   - uint64: distinct child path count.
//   - error: query or scan error, if any.
func CountImagePathByParentPath(db *sql.DB, c context.Context, root, parentPath string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/path/byParentPath", root+"/"+parentPath, map[string]any{
		"parentPath": parentPath,
	})
	q := NewQueries(db)
	qty, err := q.CountImagePathByParentPath(ctx, root, parentPath)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImageByParentPath counts images under a parent path.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root: filesystem root to search.
//   - parentPath: parent path or path prefix.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByParentPath(db *sql.DB, c context.Context, root, parentPath string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byParentPath", root+"/"+parentPath, map[string]any{
		"root":       root,
		"parentPath": parentPath,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByParentPath(ctx, root, parentPath)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImageByRoot counts images under a filesystem root.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - root: filesystem root to count.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByRoot(db *sql.DB, c context.Context, root string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byRoot", root, map[string]any{
		"root": root,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByRoot(ctx, root)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// QueryImageByTagACLPaged reads tagged visible images and attaches their tags.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images with tags.
//   - error: image query, tag query, scan, or row iteration error.
func QueryImageByTagACLPaged(db *sql.DB, c context.Context, tagID dbo.TagID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/byTag/ACL/paged", tagID, map[string]any{
		"tag":  tagID,
		"acl":  acl,
		"from": from,
		"qty":  qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageByTagACLPaged(ctx, tagID, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	for i, img := range images {
		tags, err := q.QueryTagsByImageID(ctx, *img.ID)
		if err != nil {
			logging.ExitErr(logScope, err)
			return images, err
		}
		img.Tags = dbo.TagsToPointer(tags)
		images[i] = img
	}

	logging.Exit(logScope, "ok", map[string]any{
		"found": len(images),
	})
	return images, nil
}

// CountImageByTagACL counts images bound to a tag and visible through ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByTagACL(db *sql.DB, c context.Context, tagID dbo.TagID, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byTag/Acl", tagID, map[string]any{
		"tag": tagID,
		"acl": acl,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByTagACL(ctx, tagID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// GetImageIdByTagACLFirstHash reads the first visible image ID for a tag ordered by file hash.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - dbo.ImageID: first matching image ID by file hash order.
//   - error: wrapped not-found, query, or scan error.
func GetImageIdByTagACLFirstHash(db *sql.DB, c context.Context, tagID dbo.TagID, acl dbo.ACLContext) (dbo.ImageID, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byTag/ACL/firstHash", tagID, map[string]any{
		"tag": tagID,
		"acl": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByTagACLFirstHash(ctx, tagID, acl)
	return i, returnWrapNotFound(logScope, err, "image")
}

// CountImageRoots counts distinct image roots with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - uint64: distinct root count.
//   - error: query or scan error, if any.
func CountImageRoots(db *sql.DB, c context.Context) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/roots", nil, nil)
	q := NewQueries(db)
	qty, err := q.CountImageRoots(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil

}

// QueryImageRoots reads distinct image roots with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - []string: distinct root names.
//   - error: query, scan, or row iteration error.
func QueryImageRoots(db *sql.DB, c context.Context) ([]string, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/roots", nil, nil)
	q := NewQueries(db)
	paths, err := q.QueryImageRoots(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(paths),
	})
	return paths, nil
}

// QueryImageIDByTagACLNext reads next image IDs and titles for a tag under ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - imgID: current image ID used as an ordering cursor.
//   - takenDate: current image date used as an ordering cursor.
//   - fileName: current image filename used as an ordering cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: next image IDs with display titles.
//   - error: query, scan, or row iteration error.
func QueryImageIDByTagACLNext(db *sql.DB, c context.Context, tagID dbo.TagID, imgID dbo.ImageID, takenDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byTag/ACL/next", tagID, map[string]any{
		"image_id": imgID,
		"tag_id":   tagID,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByTagACLNext(ctx, tagID, imgID, takenDate, fileName, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil

}

// QueryImageIDByTagACLPrev reads previous image IDs and titles for a tag under ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - imgID: current image ID used as an ordering cursor.
//   - takenDate: current image date used as an ordering cursor.
//   - fileName: current image filename used as an ordering cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: previous image IDs with display titles.
//   - error: query, scan, or row iteration error.
func QueryImageIDByTagACLPrev(db *sql.DB, c context.Context, tagID dbo.TagID, imgID dbo.ImageID, takenDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byTag/ACL/prev", tagID, map[string]any{
		"image_id": imgID,
		"tag_id":   tagID,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByTagACLPrev(ctx, tagID, imgID, takenDate, fileName, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil
}

// QueryImageCoordByTagByACL reads visible image coordinates for a tag.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - tagID: tag ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - []dbo.ImageCoord: image IDs, titles, and coordinates.
//   - error: query, scan, or row iteration error.
func QueryImageCoordByTagByACL(db *sql.DB, c context.Context, tagID dbo.TagID, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/coord/byTag/ACL", tagID, map[string]any{
		"tag_id": tagID,
		"acl":    acl,
	})
	q := NewQueries(db)
	imagesCoords, err := q.QueryImageCoordByTagACL(ctx, tagID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesCoords),
	})
	return imagesCoords, nil
}

// QueryImageRandomByACL reads visible images around a hash value.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - hash: file hash pivot.
//   - acl: ACL context used to build the visibility filter.
//   - qty: desired number of images.
//
// Output:
//   - []dbo.Image: matching images.
//   - error: forward or backward query error.
func QueryImageRandomByACL(db *sql.DB, c context.Context, hash string, acl dbo.ACLContext, qty int) ([]dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/random/ACL", nil, map[string]any{
		"hash": hash,
		"acl":  acl,
		"qty":  qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageRandomByHashByACLForward(ctx, hash, acl, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	newQty := qty - len(images)
	if newQty > 0 {
		images2, err := q.QueryImageRandomByHashByACLBackward(ctx, hash, acl, newQty)
		if err != nil {
			logging.ExitErr(logScope, err)
			return nil, err
		}
		images = append(images, images2...)
	}

	logging.Exit(logScope, "ok", map[string]any{
		"found": len(images),
	})
	return images, nil
}

// CountImageByLastNotSeen counts images not seen in a sync run.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - lastSyncID: sync run used as the current seen marker.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByLastNotSeen(db *sql.DB, c context.Context, lastSyncID dbo.SyncRunID) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byLastNotSeen", lastSyncID, map[string]any{
		"lastSync": lastSyncID,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByLastNotSeen(ctx, lastSyncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImageByLastSeen counts images seen in a sync run.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - lastSyncID: sync run used as the seen marker.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByLastSeen(db *sql.DB, c context.Context, lastSyncID dbo.SyncRunID) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byLastSeen", lastSyncID, map[string]any{
		"lastSync": lastSyncID,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByLastSeen(ctx, lastSyncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImage counts all images with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - uint64: image count.
//   - error: query or scan error, if any.
func CountImage(db *sql.DB, c context.Context) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count", nil, nil)
	q := NewQueries(db)
	qty, err := q.CountImage(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImageACLLevels counts images grouped by ACL level with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - dbo.ImageACLCount: counts per ACL level.
//   - error: query, scan, or row iteration error.
func CountImageACLLevels(db *sql.DB, c context.Context) (dbo.ImageACLCount, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/ACLLevels", nil, nil)
	q := NewQueries(db)
	qty, err := q.CountImageACLLevels(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return dbo.ImageACLCount{}, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// QueryImageByAlbumACLPaged reads visible images in an album and attaches their tags.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - album: album ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.Image: matching images with tags.
//   - error: image query, tag query, scan, or row iteration error.
func QueryImageByAlbumACLPaged(db *sql.DB, c context.Context, album dbo.AlbumID, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/byAlbum/ACL/paged", album, map[string]any{
		"album": album,
		"acl":   acl,
		"from":  from,
		"qty":   qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageByAlbumACLPaged(ctx, album, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	for i, img := range images {
		tags, err := q.QueryTagsByImageID(ctx, *img.ID)
		if err != nil {
			logging.ExitErr(logScope, err)
			return images, err
		}
		img.Tags = dbo.TagsToPointer(tags)
		images[i] = img
	}

	logging.Exit(logScope, "ok", map[string]any{
		"found": len(images),
	})
	return images, nil
}

// CountImageByAlbumACL counts visible images in an album.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - album: album ID to filter by.
//   - acl: ACL context used to build the visibility filter.
//
// Output:
//   - uint64: matching image count.
//   - error: query or scan error, if any.
func CountImageByAlbumACL(db *sql.DB, c context.Context, album dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byAlbum/Acl", album, map[string]any{
		"album": album,
		"acl":   acl,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByAlbumACL(ctx, album, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

// CountImageByAlbumDescendantIDsByACL counts distinct images in descendant albums visible through ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: root album ID whose ancestor list is matched.
//   - acl: ACL context used for album and image visibility filters.
//
// Output:
//   - uint64: distinct visible image count.
//   - error: query or scan error, if any.
func CountImageByAlbumDescendantIDsByACL(db *sql.DB, c context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/album/count/images/descendant/acl", nil, map[string]any{
		"acl":      acl,
		"album_id": albumID,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByAlbumDescendantIDsByACL(ctx, albumID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}
func QueryImageCoordByAlbumDescendantIDsByACL(db *sql.DB, c context.Context, albumID dbo.AlbumID, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/coord/byAlbumDescantes/ACL", albumID, map[string]any{
		"album_id": albumID,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesCoords, err := q.QueryImageCoordByAlbumDescendantIDsByACL(ctx, albumID, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesCoords),
	})
	return imagesCoords, nil
}
func QueryImageCoordByAlbumRootByACL(db *sql.DB, c context.Context, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/coord/byAlbumRoot/ACL", nil, map[string]any{
		"acl": acl,
	})
	q := NewQueries(db)
	imagesCoords, err := q.QueryImageCoordByAlbumRootByACL(ctx, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesCoords),
	})
	return imagesCoords, nil
}

// QueryImageIDByAlbumACLNext reads next image IDs and titles in an album under ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to filter by.
//   - imgID: current image ID used as a position cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: next image IDs with display titles.
//   - error: query, scan, or row iteration error.
func QueryImageIDByAlbumACLNext(db *sql.DB, c context.Context, albumID dbo.AlbumID, imgID dbo.ImageID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byAlbum/ACL/next", albumID, map[string]any{
		"album_id": albumID,
		"image_id": imgID,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByAlbumACLNext(ctx, albumID, imgID, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil
}

// QueryImageIDByAlbumACLPrev reads previous image IDs and titles in an album under ACL.
//
// Input:
//   - db: database handle.
//   - c: request context.
//   - albumID: album ID to filter by.
//   - imgID: current image ID used as a position cursor.
//   - acl: ACL context used to build the visibility filter.
//   - from: first row offset.
//   - qty: maximum number of rows.
//
// Output:
//   - []dbo.ImageTitle: previous image IDs with display titles.
//   - error: query, scan, or row iteration error.
func QueryImageIDByAlbumACLPrev(db *sql.DB, c context.Context, albumID dbo.AlbumID, imgID dbo.ImageID, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byAlbum/ACL/prev", albumID, map[string]any{
		"album_id": albumID,
		"image_id": imgID,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByAlbumACLPrev(ctx, albumID, imgID, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil
}

// CountImageByAlbumsBinding counts images with and without album bindings with logging.
//
// Input:
//   - db: database handle.
//   - c: request context.
//
// Output:
//   - uint64: number of images with at least one album binding.
//   - uint64: number of images without album binding.
//   - error: query or scan error, if any.
func CountImageByAlbumsBinding(db *sql.DB, c context.Context) (uint64, uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byAlbumsBinding", nil, nil)
	q := NewQueries(db)
	withAlbum, withoutAlbum, err := q.CountImageByAlbumsBinding(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"with_album":    withAlbum,
		"without_album": withoutAlbum,
	})
	return withAlbum, withoutAlbum, nil
}

func PatchImage(db *sql.DB, c context.Context, i dbo.Image) error {
	logScope, ctx := logging.Enter(c, "dao/image/patch", i.ID, map[string]any{"image": i})
	if i.ID == nil {
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
	if err := q.PatchImage(ctx, i); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}
