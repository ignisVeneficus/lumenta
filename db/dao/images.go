package dao

import (
	"context"
	"fmt"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"

	"database/sql"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const imageFields = `
i.id, i.path, i.filename, i.ext,
i.file_size, i.mtime, i.file_hash, i.meta_hash,
i.title, i.subject,
i.taken_at, i.camera, i.lens, i.focal_length, i.aperture, i.exposure, i.iso,
i.latitude, i.longitude, i.rotation, i.rating,
i.focus_x, i.focus_y, i.focus_mode,
i.exif_json,
i.acl_scope, i.acl_user_id, i.acl_group_id,
i.created_at, i.updated_at, i.last_seen_sync
`
const getImageById = `SELECT ` + imageFields + ` FROM images i WHERE i.id=?`

const getImageByIdACL = `SELECT ` + imageFields + ` FROM images i WHERE i.id=? and ` + aclImageWhereClause

const createImage = `
INSERT INTO images (
  path, filename, ext,
  file_size, mtime, file_hash, meta_hash,
  title, subject,
  taken_at, camera, lens, focal_length, aperture, exposure,iso,
  latitude, longitude, rotation, rating,
  focus_x, focus_y, focus_mode,
  exif_json,
  acl_scope, acl_user_id, acl_group_id, last_seen_sync
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

const deleteImage = `DELETE FROM images WHERE id=?`

const updateImage = `
UPDATE images SET
  path=?, filename=?, ext=?,
  file_size=?, mtime=?, file_hash=?, meta_hash=?,
  title=?, subject=?,
  taken_at=?, camera=?, lens=?, focal_length=?, aperture=?, exposure=?, iso=?,
  latitude=?, longitude=?, rotation=?, rating=?,
  focus_x=?, focus_y=?, focus_mode=?,
  exif_json=?,
  acl_scope=?, acl_user_id=?, acl_group_id=?, last_seen_sync=?
WHERE id=?`

const bindImageTag = `INSERT IGNORE INTO image_tags (image_id, tag_id) VALUES (?,?)`
const breakImageTag = `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`
const breakImageAllTag = `DELETE FROM image_tags WHERE image_id = ?`

const getImageByPath = `
SELECT ` + imageFields + `FROM images i WHERE i.path = ? AND i.filename = ?`

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

func parseImage(row *sql.Row) (dbo.Image, error) {
	var i dbo.Image
	err := row.Scan(
		&i.ID,
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

func (q *Queries) GetImageByPath(ctx context.Context, path, filename string) (dbo.Image, error) {
	row := q.db.QueryRowContext(ctx, getImageByPath, path, filename)
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
	log.Logger.Debug().Str("path", i.Path).Str("filename", i.Filename).Str("ext", i.Ext).Msg("Create Or Update Image")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if i.ID == nil {
		if err := q.CreateImage(ctx, *i); err != nil {
			return 0, err
		}
		id, err := q.GetLastId(ctx)
		if err != nil {
			return 0, err
		}
		i.ID = utils.PtrUint64(uint64(id))
	} else {
		if err := q.UpdateImage(ctx, *i); err != nil {
			return *i.ID, err
		}
	}
	q.BreakImageAllTag(ctx, *i.ID)
	for _, tag := range i.Tags {
		if err := writeTagTree(q, ctx, tag, *i.ID); err != nil {
			return *i.ID, err
		}
	}
	return *i.ID, tx.Commit()
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
	log.Logger.Debug().Uint64("id", id).Msg("Get image by id")

	q := NewQueries(db)
	i, err := q.GetImageById(ctx, id)
	err = wrapNotFound(err, "image")
	if err != nil {
		return i, err
	}
	tags, err := q.QuertyTagsByImageID(ctx, *i.ID)
	if err != nil {
		return i, err
	}
	tm := make(map[uint64]*dbo.Tag, len(tags))
	i.Tags = i.Tags[:0]
	for idx := range tags {
		t := &tags[idx]
		tm[*t.ID] = t
		t.Children = t.Children[:0]
	}
	for idx := range tags {
		t := &tags[idx]
		if t.ParentID == nil {
			i.Tags = append(i.Tags, *t)
			continue
		}
		if p := tm[*t.ParentID]; p != nil {
			p.Children = append(p.Children, *t)
		}
	}
	return i, nil
}
func GetImageByIdACLWTags(db *sql.DB, ctx context.Context, id uint64, acl ACLContext) (dbo.Image, error) {
	logging.Enter(ctx, "dao.image.getImageById.ACL.Tags", nil)
	log.Logger.Debug().Uint64("id", id).Msg("Get image With Tags by id with ACL")

	q := NewQueries(db)
	i, err := q.GetImageByIdACL(ctx, id, acl)
	err = wrapNotFound(err, "image")
	if err != nil {
		return i, err
	}
	tags, err := q.QuertyTagsByImageID(ctx, *i.ID)
	if err != nil {
		return i, err
	}
	tm := make(map[uint64]*dbo.Tag, len(tags))
	i.Tags = i.Tags[:0]
	for idx := range tags {
		t := &tags[idx]
		tm[*t.ID] = t
		t.Children = t.Children[:0]
	}
	for idx := range tags {
		t := &tags[idx]
		if t.ParentID == nil {
			i.Tags = append(i.Tags, *t)
			continue
		}
		if p := tm[*t.ParentID]; p != nil {
			p.Children = append(p.Children, *t)
		}
	}
	return i, nil
}

func GetImageByPath(db *sql.DB, ctx context.Context, path, filename string) (dbo.Image, error) {
	log.Logger.Debug().Str("path", path).Str("filename", filename).Msg("Get image by path")
	q := NewQueries(db)
	i, err := q.GetImageByPath(ctx, path, filename)
	return i, wrapNotFound(err, "image")
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
	stack := []*dbo.Tag{&rootTag}
	for len(stack) > 0 {
		tag := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if tag.ID == nil {
			id, err := CreateTagTx(q, ctx, *tag)
			if err != nil {
				return err
			}
			if err := q.BindImageTag(ctx, imageId, id); err != nil {
				return err
			}
			for i := len(tag.Children) - 1; i >= 0; i-- {
				c := &tag.Children[i]
				if c.ID != nil && c.ParentID != nil {
					continue
				}
				c.ParentID = &id
				stack = append(stack, c)
			}
		}
	}
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
