package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"

	"database/sql"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const imageFields = `
i.id, i.root, i.path, i.filename, i.ext,
i.file_size, i.mtime, i.file_hash, i.meta_hash,
i.title, i.subject,
i.taken_at, i.camera, i.lens, i.focal_length, i.aperture, i.exposure, i.iso,
i.latitude, i.longitude, i.rotation, i.rating, i.width, i.height, i.panorama,
i.focus_x, i.focus_y, i.focus_mode,
i.exif_json,
i.acl_scope, i.acl_user_id, i.acl_group_id,
i.created_at, i.updated_at, i.last_seen_sync
`

const defaultImageOrderForwald = ` ORDER BY order_date ASC, filename ASC, id ASC`
const defaultImageOrderBackward = ` ORDER BY order_date DESC, filename DESC, id DESC`

const getImageById = `SELECT ` + imageFields + ` FROM images i WHERE i.id=?`

const getImageByIdACL = `SELECT ` + imageFields + ` FROM images i WHERE i.id=? and ` + aclImageWhereClause

const createImage = `
INSERT INTO images (
  root, path, filename, ext,
  file_size, mtime, file_hash, meta_hash,
  title, subject,
  taken_at, camera, lens, focal_length, aperture, exposure,iso,
  latitude, longitude, rotation, rating, width, height, panorama,
  focus_x, focus_y, focus_mode,
  exif_json,
  acl_scope, acl_user_id, acl_group_id, last_seen_sync
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

const deleteImage = `DELETE FROM images WHERE id=?`

const updateImage = `
UPDATE images SET
  root=?, path=?, filename=?, ext=?,
  file_size=?, mtime=?, file_hash=?, meta_hash=?,
  title=?, subject=?,
  taken_at=?, camera=?, lens=?, focal_length=?, aperture=?, exposure=?, iso=?,
  latitude=?, longitude=?, rotation=?, rating=?, width=?, height=?, panorama=?,
  focus_x=?, focus_y=?, focus_mode=?,
  exif_json=?,
  acl_scope=?, acl_user_id=?, acl_group_id=?, last_seen_sync=?
WHERE id=?`

const bindImageTag = `INSERT IGNORE INTO image_tags (image_id, tag_id) VALUES (?,?)`
const breakImageTag = `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`
const breakImageAllTag = `DELETE FROM image_tags WHERE image_id = ?`

const getImageByPath = `
SELECT ` + imageFields + `FROM images i WHERE i.root=? AND i.path = ? AND i.filename = ?`

const countImagesByAlbums = `
SELECT COUNT(DISTINCT ai.image_id)
FROM album_images ai
JOIN images i ON i.id = ai.image_id
WHERE ai.album_id IN (%s)
` + aclImageWhereClause

const queryImageIDsByAlbumsPage = `
SELECT ai.image_id
FROM album_images ai
JOIN images i ON i.id = ai.image_id
WHERE ai.album_id IN (%s)
AND ai.image_id > ?
` + aclImageWhereClause + `
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

const imageByTagACLWhere = ` FROM images AS i JOIN image_tags AS it ON i.id = it.image_id WHERE it.tag_id = ? AND ` + aclImageWhereClause
const queryImageByTagACLPaged = `SELECT ` + imageFields + imageByTagACLWhere + defaultImageOrderForwald + ` LIMIT ?, ? `
const countImageByTagACL = `SELECT COUNT(*) ` + imageByTagACLWhere

const getImageIdByTagACLFirstHash = `SELECT i.id ` + imageByTagACLWhere + ` ORDER BY i.file_hash LIMIT 1 `

const countImageRoots = `SELECT COUNT(DISTINCT i.root) AS root_count FROM images i`
const queryImageRoots = `SELECT DISTINCT i.root FROM images i order by i.root`

// TODO: finish it
const queryImageByTagACLAround = ""

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
		&i.Subject,
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
		&i.ACLScope,
		&i.ACLUserID,
		&i.ACLGroupID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.LastSeenSync,
	)
	return i, err
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
			&i.Subject,
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
			&i.ACLScope,
			&i.ACLUserID,
			&i.ACLGroupID,
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

func (q *Queries) GetImageByIdACL(ctx context.Context, id uint64, acl ACLContext) (dbo.Image, error) {
	params := []any{id}
	params = append(params, acl.AsParamArray()...)
	row := q.db.QueryRowContext(ctx, getImageByIdACL, params...)
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
		i.Subject,
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
		i.ACLScope,
		i.ACLUserID,
		i.ACLGroupID,
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
		i.Subject,
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
		i.ACLScope,
		i.ACLUserID,
		i.ACLGroupID,
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

func (q *Queries) GetImageByPath(ctx context.Context, root, path, filename string) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageByPath, root, path, filename)
	return parseImage(row)
}

func (q *Queries) CountImagesByAlbums(ctx context.Context, albumIDs []uint64, acl ACLContext) (uint64, error) {
	if len(albumIDs) == 0 {
		return 0, nil
	}

	inClause, args := buildUint64InClause(albumIDs)

	query := fmt.Sprintf(countImagesByAlbums, inClause)
	params := append(args, acl.AsParamArray()...)

	row := q.db.QueryRowContext(ctx, query, params...)

	var cnt uint64
	err := row.Scan(&cnt)
	return cnt, err
}

func (q *Queries) QueryImageIDsByAlbumsPage(ctx context.Context, albumIDs []uint64, cursor uint64, pageSize uint32, acl ACLContext) ([]uint64, error) {
	if len(albumIDs) == 0 || pageSize == 0 {
		return []uint64{}, nil
	}

	inClause, args := buildUint64InClause(albumIDs)
	query := fmt.Sprintf(queryImageIDsByAlbumsPage, inClause)

	params := make([]any, 0, len(args)+len(acl.AsParamArray())+2)
	params = append(params, args...)
	params = append(params, cursor)
	params = append(params, acl.AsParamArray()...)
	params = append(params, pageSize)

	rows, err := q.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uint64, 0, pageSize)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (q *Queries) DeleteImagesNotSeen(ctx context.Context, syncID uint64, limit uint32) (uint64, error) {
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
			&i.Subject,
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
			&i.ACLScope,
			&i.ACLUserID,
			&i.ACLGroupID,
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
func (q *Queries) CountImagesByPath(ctx context.Context, root, path string) (uint64, error) {
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
func (q *Queries) countImagePathByParentPath(ctx context.Context, root, parentPath string) (uint64, error) {
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
func (q *Queries) QueryImageByTagACLPaged(ctx context.Context, tagId uint64, acl ACLContext, from, qty uint64) ([]dbo.Image, error) {
	params := []any{tagId}
	params = append(params, acl.AsParamArray()...)
	params = append(params, from)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, queryImageByTagACLPaged, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images, err := parseImageRows(rows)
	return images, err
}

func (q *Queries) CountImageByTagACL(ctx context.Context, tagId uint64, acl ACLContext) (uint64, error) {
	params := []any{tagId}
	params = append(params, acl.AsParamArray()...)
	row := q.db.QueryRowContext(ctx, countImageByTagACL, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetImageIdByTagACLFirstHash(ctx context.Context, tagId uint64, acl ACLContext) (uint64, error) {
	params := []any{tagId}
	params = append(params, acl.AsParamArray()...)
	row := q.db.QueryRowContext(ctx, getImageIdByTagACLFirstHash, params...)
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

//
// =========================================================
// Public API functions
// =========================================================
//

func BindImageTag(db *sql.DB, ctx context.Context, imageId, tagId uint64) error {
	log.Logger.Debug().Uint64("image", imageId).Uint64("tag", tagId).Msg("Bind Image Tag")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.BindImageTag(ctx, imageId, tagId); err != nil {
		return err
	}
	return tx.Commit()
}

func CreateImage(db *sql.DB, ctx context.Context, i *dbo.Image) (uint64, error) {
	log.Logger.Debug().Str("path", i.Path).Str("filename", i.Filename).Str("ext", i.Ext).Msg("Create image")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateImage(ctx, *i); err != nil {
		return 0, err
	}

	id, err := q.GetLastId(ctx)
	if err != nil {
		return 0, err
	}
	i.ID = utils.PtrUint64(uint64(id))

	return uint64(id), tx.Commit()
}
func UpdateImage(db *sql.DB, ctx context.Context, i dbo.Image) error {
	log.Logger.Debug().Uint64("id", *i.ID).Msg("Update image")

	if i.ID == nil {
		return sql.ErrNoRows
	}

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateImage(ctx, i); err != nil {
		return err
	}

	return tx.Commit()
}
func CreateOrUpdateImage(db *sql.DB, ctx context.Context, i *dbo.Image) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.createOrUpdate", map[string]any{
		"path":     i.Path,
		"filename": i.Filename,
		"ext":      i.Ext,
	})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if i.ID == nil {
		if err := q.CreateImage(ctx, *i); err != nil {
			logging.ExitErr(logg, err)
			return 0, err
		}
		id, err := q.GetLastId(ctx)
		if err != nil {
			logging.ExitErr(logg, err)
			return 0, err
		}
		i.ID = utils.PtrUint64(uint64(id))
	} else {
		if err := q.UpdateImage(ctx, *i); err != nil {
			logging.ExitErr(logg, err)
			return *i.ID, err
		}
	}
	q.BreakImageAllTag(ctx, *i.ID)
	for _, tag := range i.Tags {
		if err := writeTagTree(q, ctx, *tag, *i.ID); err != nil {
			logging.ExitErr(logg, err)
			return *i.ID, err
		}
	}
	return *i.ID, logging.Return(logg, tx.Commit())
}

func DeleteImage(db *sql.DB, ctx context.Context, id uint64) error {
	log.Logger.Debug().Uint64("id", id).Msg("Delete image")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteImage(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}
func GetImageById(db *sql.DB, ctx context.Context, id uint64) (dbo.Image, error) {
	log.Logger.Debug().Uint64("id", id).Msg("Get image by id")

	q := NewQueries(db)
	i, err := q.GetImageById(ctx, id)
	return i, wrapNotFound(err, "image")
}
func GetImageByIdACL(db *sql.DB, ctx context.Context, id uint64, acl ACLContext) (dbo.Image, error) {
	logg := logging.Enter(ctx, "dao.image.getById.ACL", map[string]any{
		"image_id":    id,
		"acl_context": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	return i, returnWrapNotFound(logg, err, "image")
}

func GetImageByIdWTags(db *sql.DB, ctx context.Context, id uint64) (dbo.Image, error) {
	logg := logging.Enter(ctx, "dao.image.getImageById.Tags", nil)
	q := NewQueries(db)
	i, err := q.GetImageById(ctx, id)
	err = wrapNotFound(err, "image")
	if err != nil {
		logging.ExitErr(logg, err)
		return i, err
	}
	tags, err := q.QueryTagsByImageID(ctx, *i.ID)
	if err != nil {
		logging.ExitErr(logg, err)
		return i, err
	}
	i.AddTags(tags)
	logging.Exit(logg, "Ok", nil)
	return i, nil
}
func GetImageByIdACLWTags(db *sql.DB, ctx context.Context, id uint64, acl ACLContext) (dbo.Image, error) {
	logg := logging.Enter(ctx, "dao.image.getImageById.ACL.Tags", nil)

	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	err = wrapNotFound(err, "image")
	if err != nil {
		logging.ExitErr(logg, err)
		return i, err
	}
	tags, err := q.QueryTagsByImageID(ctx, *i.ID)
	if err != nil {
		logging.ExitErr(logg, err)
		return i, err
	}
	i.AddTags(tags)
	logging.Exit(logg, "Ok", nil)
	return i, nil
}

func GetImageByPath(db *sql.DB, ctx context.Context, root, path, filename string) (dbo.Image, error) {
	logg := logging.Enter(ctx, "dao.image.get.byPath", map[string]any{
		"root":     root,
		"path":     path,
		"filename": filename,
	})
	q := NewQueries(db)
	i, err := q.GetImageByPath(ctx, root, path, filename)
	return i, returnWrapNotFound(logg, err, "image")
}

func CountImagesByAlbums(db *sql.DB, ctx context.Context, albumIDs []uint64, acl ACLContext) (uint64, error) {
	log.Logger.Debug().Int("album _count", len(albumIDs)).Object("acl", logging.WithLevel(zerolog.DebugLevel, &acl)).Msg("Count images by albums")
	q := NewQueries(db)
	cnt, err := q.CountImagesByAlbums(ctx, albumIDs, acl)
	return cnt, err
}

func QueryImageIDsByAlbumsPage(db *sql.DB, ctx context.Context, albumIDs []uint64, cursor uint64, pageSize uint32, acl ACLContext) ([]uint64, error) {
	log.Logger.Debug().Int("album_count", len(albumIDs)).Uint64("cursor", cursor).Uint32("pageSize", pageSize).Object("acl", logging.WithLevel(zerolog.DebugLevel, &acl)).Msg("List image IDs by albums (paged)")

	q := NewQueries(db)
	return q.QueryImageIDsByAlbumsPage(ctx, albumIDs, cursor, pageSize, acl)
}
func BindImageTags(db *sql.DB, ctx context.Context, imageID uint64, tagIDs []uint64) error {

	if len(tagIDs) == 0 {
		return nil
	}

	log.Logger.Debug().Uint64("image_id", imageID).Int("tag_count", len(tagIDs)).Msg("Bind image tags")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	for _, tagID := range tagIDs {
		if err := q.BindImageTag(ctx, imageID, tagID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func writeTagTree(q *Queries, ctx context.Context, rootTag dbo.Tag, imageId uint64) error {
	logg := logging.Enter(ctx, "dao.Image.writeTagTree", map[string]any{"root": rootTag})
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
				logging.ExitErrParams(logg, err, map[string]any{"reason": "sql-error"})
				return err
			default:
				id, err = CreateTagTx(q, ctx, *tag)
				if err != nil {
					logging.ExitErr(logg, err)
					return err
				}
			}
		}
		if err := q.BindImageTag(ctx, imageId, id); err != nil {
			logging.ExitErr(logg, err)
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
	logging.Exit(logg, "ok", nil)
	return nil
}

func DeleteImagesNotSeen(db *sql.DB, ctx context.Context, syncID uint64, limit uint32) (uint64, error) {
	log.Logger.Debug().Uint64("sync_id", syncID).Uint32("limit", limit).Msg("Delete images not seen in sync")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	deleted, err := q.DeleteImagesNotSeen(ctx, syncID, limit)
	if err != nil {
		return 0, err
	}
	return deleted, tx.Commit()
}
func DeleteImagesNotSeenAll(db *sql.DB, ctx context.Context, syncID uint64, limit uint32) error {
	log.Logger.Debug().Uint64("sync_id", syncID).Uint32("limit", limit).Msg("Delete All images not seen in sync")
	deleted, err := DeleteImagesNotSeen(db, ctx, syncID, limit)
	if err != nil {
		return err
	}
	batch := 1
	for deleted > 0 {
		log.Logger.Debug().Int("batch", batch).Uint64("sync_id", syncID).Uint32("limit", limit).Msg("Delete All images not seen in sync in batch")
		deleted, err = DeleteImagesNotSeen(db, ctx, syncID, limit)
		if err != nil {
			return err
		}
		batch++
	}
	log.Logger.Debug().Int("batch", batch).Uint64("sync_id", syncID).Uint32("limit", limit).Msg("Delete All images not seen in sync in batch")
	return nil
}

func UpdateImageSyncId(db *sql.DB, ctx context.Context, imageID uint64, syncID uint64) error {
	log.Logger.Debug().Uint64("sync_id", syncID).Uint64("image_id", imageID).Msg("Update Image Sync Id")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	err = q.UpdateImageSyncId(ctx, imageID, syncID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func QueryImageWLastSyncWUserByPathPaged(db *sql.DB, ctx context.Context, root, path string, from, qty uint64) ([]dbo.ImageWLastSyncWUser, error) {

	logg := logging.Enter(ctx, "dao.image.queryImagesWLastSyncWUserByPathPaged", map[string]any{
		"root": root,
		"path": path,
		"from": from,
		"qty":  qty,
	})

	q := NewQueries(db)

	images, err := q.QueryImageWLastSyncByWUserPathPaged(ctx, root, path, from, qty)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}

	logging.Exit(logg, "ok", map[string]any{
		"found": len(images),
	})

	return images, nil
}
func CountImagesByPath(db *sql.DB, ctx context.Context, root, path string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.ByPath", map[string]any{
		"path": path,
	})
	q := NewQueries(db)
	qty, err := q.CountImagesByPath(ctx, root, path)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil
}

func GetImageIdByPathFirstHash(db *sql.DB, ctx context.Context, root, path string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.get.id.byPath.firstHash", map[string]any{
		"root": root,
		"path": path,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByPathFirstHash(ctx, root, path)
	return i, returnWrapNotFound(logg, err, "image")
}

func GetImageIdByRootFirstHash(db *sql.DB, ctx context.Context, root string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.get.id.byRoot.firstHash", map[string]any{
		"root": root,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByRootFirstHash(ctx, root)
	return i, returnWrapNotFound(logg, err, "image")
}

func QueryImagePathByParentPathPaged(db *sql.DB, ctx context.Context, root, parentPath string, from, qty uint64) ([]string, error) {
	logg := logging.Enter(ctx, "dao.image.query.path.byParentPath.paged", map[string]any{
		"parentPpath": parentPath,
		"from":        from,
		"qty":         qty,
	})
	q := NewQueries(db)
	paths, err := q.QueryImagePathByParentPathPaged(ctx, root, parentPath, from, qty)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	logging.Exit(logg, "ok", map[string]any{
		"found": len(paths),
	})
	return paths, nil
}
func CountImagePathByParentPath(db *sql.DB, ctx context.Context, root, parentPath string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.path.byParentPath", map[string]any{
		"parentPath": parentPath,
	})
	q := NewQueries(db)
	qty, err := q.countImagePathByParentPath(ctx, root, parentPath)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil
}
func CountImageByParentPath(db *sql.DB, ctx context.Context, root, parentPath string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.byParentPath", map[string]any{
		"root":       root,
		"parentPath": parentPath,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByParentPath(ctx, root, parentPath)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil
}
func CountImageByRoot(db *sql.DB, ctx context.Context, root string) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.byRoot", map[string]any{
		"root": root,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByRoot(ctx, root)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil
}

func QueryImageByTagACLPaged(db *sql.DB, ctx context.Context, tag uint64, acl ACLContext, from, qty uint64) ([]dbo.Image, error) {
	logg := logging.Enter(ctx, "dao.image.query.byTag.byACL.paged", map[string]any{
		"tag":  tag,
		"acl":  acl,
		"from": from,
		"qty":  qty,
	})
	q := NewQueries(db)
	images, err := q.QueryImageByTagACLPaged(ctx, tag, acl, from, qty)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	for i, img := range images {
		tags, err := q.QueryTagsByImageID(ctx, *img.ID)
		if err != nil {
			logging.ExitErr(logg, err)
			return images, err
		}
		img.AddTags(tags)
		images[i] = img
	}

	logging.Exit(logg, "ok", map[string]any{
		"found": len(images),
	})
	return images, nil
}
func CountImageByTagACL(db *sql.DB, ctx context.Context, tag uint64, acl ACLContext) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.byTag.byAcl", map[string]any{
		"tag": tag,
		"acl": acl,
	})
	q := NewQueries(db)
	qty, err := q.CountImageByTagACL(ctx, tag, acl)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil
}

func GetImageIdByTagACLFirstHash(db *sql.DB, ctx context.Context, tag uint64, acl ACLContext) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.get.byTag.byACL.firstHash", map[string]any{
		"tag": tag,
		"acl": acl,
	})
	q := NewQueries(db)
	i, err := q.GetImageIdByTagACLFirstHash(ctx, tag, acl)
	return i, returnWrapNotFound(logg, err, "image")
}

func CountImageRoots(db *sql.DB, ctx context.Context) (uint64, error) {
	logg := logging.Enter(ctx, "dao.image.count.roots", nil)
	q := NewQueries(db)
	qty, err := q.CountImageRoots(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return 0, err
	}
	logging.Exit(logg, "ok", map[string]any{"return": qty})
	return qty, nil

}

func QueryImageRoots(db *sql.DB, ctx context.Context) ([]string, error) {
	logg := logging.Enter(ctx, "dao.image.query.roots", nil)
	q := NewQueries(db)
	paths, err := q.QueryImageRoots(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	logging.Exit(logg, "ok", map[string]any{
		"found": len(paths),
	})
	return paths, nil
}
