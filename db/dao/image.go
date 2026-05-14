package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"

	"database/sql"
)

const imageFields = `
i.id, i.root, i.path, i.filename, i.ext,
i.file_size, i.mtime, i.file_hash, i.meta_hash,
i.title, i.caption,
i.taken_at, i.camera, i.lens, i.focal_length, i.aperture, i.exposure, i.iso,
i.latitude, i.longitude, i.rotation, i.rating, i.width, i.height, i.panorama,
i.focus_x, i.focus_y, i.focus_mode,
i.exif_json,
i.acl_level, i.acl_user_id, 
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
  focus_x, focus_y, focus_mode,
  exif_json,
  acl_level, acl_user_id, last_seen_sync
) VALUES (?,?,?,?,?,?,?,?,?,?,?,IFNULL(taken_at, '1000-01-01 00:00:00'),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

const deleteImage = `DELETE FROM images WHERE id=?`

const updateImage = `
UPDATE images SET
  root=?, path=?, filename=?, ext=?,
  file_size=?, mtime=?, file_hash=?, meta_hash=?,
  title=?, caption=?,
  taken_at=?, order_date = IFNULL(taken_at, '1000-01-01 00:00:00'),
  camera=?, lens=?, focal_length=?, aperture=?, exposure=?, iso=?,
  latitude=?, longitude=?, rotation=?, rating=?, width=?, height=?, panorama=?,
  focus_x=?, focus_y=?, focus_mode=?,
  exif_json=?,
  acl_level=?, acl_user_id=?, last_seen_sync=?
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

const updateImageSyncId = `UPDATE images SET last_seen_sync=? WHERE id=?`

const queryImagesWLastSyncWUserByPathPaged = `SELECT ` + imageFields + ` , sr.finished_at, u.username FROM images AS i LEFT JOIN sync_runs AS sr ON i.last_seen_sync=sr.id LEFT JOIN users AS u ON u.id=i.acl_user_id WHERE i.root=? AND i.path = ? ORDER BY i.filename, i.ext LIMIT ?,?`
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
		&i.ExifJSON,
		&i.ACLLevel,
		&i.ACLUserID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.LastSeenSync,
	)
	return i, err
}

func getImageParamsNextPrev(id uint64, orderDate *time.Time, filename string) []any {
	if orderDate == nil {
		// default from database
		oD, _ := time.Parse("2006-01-02 15:04:05", "1000-01-01 00:00:00")
		orderDate = &oD
	}
	return []any{
		orderDate, orderDate, filename, orderDate, filename, id,
	}
}

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
			&i.ExifJSON,
			&i.ACLLevel,
			&i.ACLUserID,
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

func (q *Queries) GetImageById(ctx context.Context, id uint64) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageById, id)
	return parseImage(row)
}

func (q *Queries) GetImageByIdACL(ctx context.Context, id uint64, acl dbo.ACLContext) (dbo.Image, error) {
	sql, params := CreateAclWhere("i", acl)
	params = append([]any{id}, params...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(getImageByIdACL, sql), params...)
	return parseImage(row)
}

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
		i.ExifJSON,
		i.ACLLevel,
		i.ACLUserID,
		i.LastSeenSync,
	)
	return err
}

func (q *Queries) DeleteImage(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteImage, id)
	return err
}
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
		i.ExifJSON,
		i.ACLLevel,
		i.ACLUserID,
		i.LastSeenSync,
		i.ID,
	)
	return err
}

func (q *Queries) BindImageTag(ctx context.Context, imageId, tagId uint64) error {
	_, err := q.db.ExecContext(ctx, bindImageTag, imageId, tagId)
	return err
}

func (q *Queries) BreakImageTag(ctx context.Context, imageId, tagId uint64) error {
	_, err := q.db.ExecContext(ctx, breakImageTag, imageId, tagId)
	return err
}
func (q *Queries) BreakImageAllTag(ctx context.Context, imageId uint64) error {
	_, err := q.db.ExecContext(ctx, breakImageAllTag, imageId)
	return err
}

func (q *Queries) GetImageByPath(ctx context.Context, root, path, filename, ext string) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageByPath, root, path, filename, ext)
	return parseImage(row)
}

func (q *Queries) CountImageByAlbums(ctx context.Context, albumIDs []uint64, acl dbo.ACLContext) (uint64, error) {
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

func (q *Queries) QueryImageIDsByAlbumsPage(ctx context.Context, albumIDs []uint64, acl dbo.ACLContext, from, qty uint64) ([]uint64, error) {
	if len(albumIDs) == 0 || qty == 0 {
		return []uint64{}, nil
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

	ids := make([]uint64, 0, qty)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (q *Queries) DeleteImageNotSeen(ctx context.Context, syncID uint64, limit uint32) (uint64, error) {
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
func (q *Queries) UpdateImageSyncId(ctx context.Context, imageId uint64, syncId uint64) error {
	_, err := q.db.ExecContext(ctx, updateImageSyncId, syncId, imageId)
	return err
}

func (q *Queries) QueryImageWLastSyncByWUserPathPaged(ctx context.Context, root, path string, from, qty uint64) ([]dbo.ImageWLastSyncWUser, error) {
	rows, err := q.db.QueryContext(ctx, queryImagesWLastSyncWUserByPathPaged, root, path, from, qty)
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
			&i.ExifJSON,
			&i.ACLLevel,
			&i.ACLUserID,
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
func (q *Queries) CountImageByPath(ctx context.Context, root, path string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImagesByPath, root, path)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetImageIdByPathFirstHash(ctx context.Context, root, path string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, getImageIdByPathFirstHash, root, path, path)
	var id uint64
	err := row.Scan(&id)
	return id, err
}
func (q *Queries) GetImageIdByRootFirstHash(ctx context.Context, root string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, getImageIdByRootFirstHash, root)
	var id uint64
	err := row.Scan(&id)
	return id, err
}

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

func (q *Queries) CountImageByParentPath(ctx context.Context, root, parentPath string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByParentPath, root, parentPath, parentPath)
	var count uint64
	err := row.Scan(&count)
	return count, err
}
func (q *Queries) CountImageByRoot(ctx context.Context, root string) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByRoot, root)
	var count uint64
	err := row.Scan(&count)
	return count, err
}
func (q *Queries) QueryImageByTagACLPaged(ctx context.Context, tagId uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagId}
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

func (q *Queries) CountImageByTagACL(ctx context.Context, tagId uint64, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagId}
	params = append(params, aclParams...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(countImageByTagACL, aclWhere), params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetImageIdByTagACLFirstHash(ctx context.Context, tagId uint64, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{tagId}
	params = append(params, aclParams...)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(getImageIdByTagACLFirstHash, aclWhere), params...)
	var id uint64
	err := row.Scan(&id)
	return id, err
}

func (q *Queries) CountImageRoots(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageRoots)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

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

func (q *Queries) QueryImageIDByTagACLNext(ctx context.Context, tagId uint64, imgId uint64, orderDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagId}
	params = append(params, aclParams...)
	params = append(params, getImageParamsNextPrev(imgId, orderDate, fileName)...)
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

func (q *Queries) QueryImageIDByTagACLPrev(ctx context.Context, tagId uint64, imgId uint64, orderDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagId}
	params = append(params, aclParams...)
	params = append(params, getImageParamsNextPrev(imgId, orderDate, fileName)...)
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

func (q *Queries) QueryImageCoordByTagACL(ctx context.Context, tagId uint64, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)

	params := []any{tagId}
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

func (q *Queries) CountImageByLastNotSeen(ctx context.Context, lastSyncId uint64) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByLastNotSeen, lastSyncId)
	var count uint64
	err := row.Scan(&count)
	return count, err
}
func (q *Queries) CountImageByLastSeen(ctx context.Context, lastSyncId uint64) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImageByLastSeen, lastSyncId)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) CountImage(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, countImage)
	var count uint64
	err := row.Scan(&count)
	return count, err
}
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

func (q *Queries) QueryImageByAlbumACLPaged(ctx context.Context, albumId uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
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

func (q *Queries) CountImageByAlbumACL(ctx context.Context, albumId uint64, acl dbo.ACLContext) (uint64, error) {
	aclWhere, aclParams := CreateAclWhere("i", acl)
	params := []any{}
	params = append(params, aclParams...)
	params = append(params, albumId)
	row := q.db.QueryRowContext(ctx, fmt.Sprintf(countImageByAlbumACL, aclWhere), params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

//
// =========================================================
// Public API functions
// =========================================================
//

func BindImageTag(db *sql.DB, c context.Context, imageId, tagId uint64) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/bind", tagId, map[string]any{
		"image_id": imageId,
		"tag_id":   tagId,
	})
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindImageTag(ctx, imageId, tagId); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

func CreateImage(db *sql.DB, c context.Context, i *dbo.Image) (uint64, error) {
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
	i.ID = utils.PtrUint64(uint64(id))

	return uint64(id), logging.ReturnParams(logScope, tx.Commit(), map[string]any{"new_id": id})
}
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
func CreateOrUpdateImage(db *sql.DB, c context.Context, i *dbo.Image) (uint64, error) {
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
		i.ID = utils.PtrUint64(uint64(id))
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

func DeleteImage(db *sql.DB, c context.Context, id uint64) error {
	logScope, ctx := logging.Enter(c, "dao/image/delete", id, map[string]any{
		"image_id": id,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteImage(ctx, id); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}
func GetImageById(db *sql.DB, c context.Context, id uint64) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byId", id, map[string]any{
		"image_id": id,
	})

	q := NewQueries(db)
	i, err := q.GetImageById(ctx, id)
	return i, returnWrapNotFound(logScope, err, "image")
}
func GetImageByIdACL(db *sql.DB, c context.Context, id uint64, acl dbo.ACLContext) (dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byId/ACL", id, map[string]any{
		"image_id":    id,
		"acl_context": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	return i, returnWrapNotFound(logScope, err, "image")
}

func GetImageByIdWTags(db *sql.DB, c context.Context, id uint64) (dbo.Image, error) {
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
func GetImageByIdACLWTags(db *sql.DB, c context.Context, id uint64, acl dbo.ACLContext) (dbo.Image, error) {
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

func CountImageByAlbums(db *sql.DB, c context.Context, albumIDs []uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byAlbums/ACL", nil, map[string]any{
		"acl":    acl,
		"albums": albumIDs,
	})
	q := NewQueries(db)
	cnt, err := q.CountImageByAlbums(ctx, albumIDs, acl)
	return cnt, logging.Return(logScope, err)
}

func QueryImageIDsByAlbumsPage(db *sql.DB, c context.Context, albumIDs []uint64, acl dbo.ACLContext, from, qty uint64) ([]uint64, error) {
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
func BindImageTags(db *sql.DB, c context.Context, imageID uint64, tagIDs []uint64) error {
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

func writeTagTree(q *Queries, c context.Context, rootTag dbo.Tag, imageId uint64) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/tag/tagTree", imageId, map[string]any{"root": rootTag})
	stack := []*dbo.Tag{&rootTag}
	for len(stack) > 0 {
		tag := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		var id uint64
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

func DeleteImageNotSeen(db *sql.DB, c context.Context, syncID uint64, limit uint32) (uint64, error) {
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
func DeleteImageNotSeenAll(db *sql.DB, c context.Context, syncID uint64, limit uint32) error {
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

func UpdateImageSyncId(db *sql.DB, c context.Context, imageID uint64, syncID uint64) error {
	logScope, ctx := logging.Enter(c, "dao/image/update/syncId", imageID, map[string]any{
		"image_id": imageID,
		"sync_id":  syncID})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	err = q.UpdateImageSyncId(ctx, imageID, syncID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

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

func GetImageIdByPathFirstHash(db *sql.DB, c context.Context, root, path string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/id/byPath/firstHash", root+"/"+path, map[string]any{
		"root": root,
		"path": path,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByPathFirstHash(ctx, root, path)
	return i, returnWrapNotFound(logScope, err, "image")
}

func GetImageIdByRootFirstHash(db *sql.DB, c context.Context, root string) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/id/byRoot/firstHash", root, map[string]any{
		"root": root,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByRootFirstHash(ctx, root)
	return i, returnWrapNotFound(logScope, err, "image")
}

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

func QueryImageByTagACLPaged(db *sql.DB, c context.Context, tag uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/byTag/ACL/paged", tag, map[string]any{
		"tag":  tag,
		"acl":  acl,
		"from": from,
		"qty":  qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageByTagACLPaged(ctx, tag, acl, from, qty)
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
func CountImageByTagACL(db *sql.DB, c context.Context, tag uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/count/byTag/Acl", tag, map[string]any{
		"tag": tag,
		"acl": acl,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByTagACL(ctx, tag, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return 0, err
	}
	logging.Exit(logScope, "ok", map[string]any{"return": qty})
	return qty, nil
}

func GetImageIdByTagACLFirstHash(db *sql.DB, c context.Context, tag uint64, acl dbo.ACLContext) (uint64, error) {
	logScope, ctx := logging.Enter(c, "dao/image/get/byTag/ACL/firstHash", tag, map[string]any{
		"tag": tag,
		"acl": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByTagACLFirstHash(ctx, tag, acl)
	return i, returnWrapNotFound(logScope, err, "image")
}

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

func QueryImageIDByTagACLNext(db *sql.DB, c context.Context, tagId uint64, imgId uint64, takenDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byTag/ACL/next", tagId, map[string]any{
		"image_id": imgId,
		"tag_id":   tagId,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByTagACLNext(ctx, tagId, imgId, takenDate, fileName, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil

}

func QueryImageIDByTagACLPrev(db *sql.DB, c context.Context, tagId uint64, imgId uint64, takenDate *time.Time, fileName string, acl dbo.ACLContext, from, qty uint64) ([]dbo.ImageTitle, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/idTitle/byTag/ACL/prev", tagId, map[string]any{
		"image_id": imgId,
		"tag_id":   tagId,
		"from":     from,
		"qty":      qty,
		"acl":      acl,
	})
	q := NewQueries(db)
	imagesTitle, err := q.QueryImageIDByTagACLPrev(ctx, tagId, imgId, takenDate, fileName, acl, from, qty)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesTitle),
	})
	return imagesTitle, nil
}

func QueryImageCoordByTagByACL(db *sql.DB, c context.Context, tagId uint64, acl dbo.ACLContext) ([]dbo.ImageCoord, error) {
	logScope, ctx := logging.Enter(c, "dao/image/query/coord/byTag/ACL", tagId, map[string]any{
		"tag_id": tagId,
		"acl":    acl,
	})
	q := NewQueries(db)
	imagesCoords, err := q.QueryImageCoordByTagACL(ctx, tagId, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	logging.Exit(logScope, "ok", map[string]any{
		"found": len(imagesCoords),
	})
	return imagesCoords, nil
}
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
		images2, err := q.QueryImageRandomByHashByACLBackward(ctx, hash, acl, qty)
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
func CountImageByLastNotSeen(db *sql.DB, c context.Context, lastSyncID uint64) (uint64, error) {
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

func CountImageByLastSeen(db *sql.DB, c context.Context, lastSyncID uint64) (uint64, error) {
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

func QueryImageByAlbumACLPaged(db *sql.DB, c context.Context, album uint64, acl dbo.ACLContext, from, qty uint64) ([]dbo.Image, error) {
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
func CountImageByAlbumACL(db *sql.DB, c context.Context, album uint64, acl dbo.ACLContext) (uint64, error) {
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
