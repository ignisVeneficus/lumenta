package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const tagFields = `t.id, t.name, t.parent_id, t.source`

const getTagByID = `SELECT ` + tagFields + ` FROM tags t WHERE t.id = ?`

const getTagByParentAndName = `SELECT ` + tagFields + ` FROM tags t
WHERE t.parent_id <=> ?
  AND t.name = ?
`

const listTagsByParent = `SELECT ` + tagFields + ` FROM tags t
WHERE t.parent_id <=> ?
ORDER BY t.name
`

const createTag = `INSERT INTO tags 
(name, parent_id, source) VALUES (?,?,?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id);
`

const deleteTag = `DELETE FROM tags WHERE id = ?`

const queryTagsByImageID = `SELECT ` + tagFields + ` FROM image_tags it
JOIN tags t ON t.id = it.tag_id
WHERE it.image_id = ?
ORDER BY t.name
`

const queryTags = `SELECT ` + tagFields + ` FROM tags t `

const queryTagsByACL = `SELECT
    t.id,
    t.name,
    t.parent_id,
    t.source,
    COUNT(*) AS image_count
FROM image_tags AS it
JOIN tags AS t ON t.id = it.tag_id
JOIN images AS i ON i.id = it.image_id
WHERE ` + aclImageWhereClause + `
GROUP BY
    t.id,
    t.name,
    t.parent_id,
    t.source
`

const queryTagsByParentACLPaged = `SELECT
    t.id,
    t.name,
    t.parent_id,
    t.source,
    COUNT(*) AS image_count
FROM image_tags it
JOIN tags t   ON t.id = it.tag_id
JOIN images i ON i.id = it.image_id
WHERE t.parent_id=? AND ` +
	aclImageWhereClause + `
GROUP BY
    t.id,
    t.name,
    t.parent_id,
    t.source
ORDER BY t.name
LIMIT ?, ?	
`
const countTagsByParentACL = `SELECT COUNT(DISTINCT t.id)
FROM image_tags it
JOIN tags t   ON t.id = it.tag_id
JOIN images i ON i.id = it.image_id
WHERE t.parent_id = ?
  AND ` + aclImageWhereClause

const getTagByIDACL = ` SELECT ` + tagFields + ` FROM tags as t 
WHERE t.id = ?
  AND EXISTS (
      SELECT 1
      FROM image_tags it
      JOIN images i ON i.id = it.image_id
      WHERE it.tag_id = t.id
        AND ` + aclImageWhereClause + ` 
);`

func parseTagRow(row *sql.Row) (dbo.Tag, error) {
	var t dbo.Tag
	err := row.Scan(&t.ID, &t.Name, &t.ParentID, &t.Source)
	return t, err
}

func parseTagRows(rows *sql.Rows) ([]dbo.Tag, error) {
	defer rows.Close()

	var out []dbo.Tag
	for rows.Next() {
		var t dbo.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.ParentID, &t.Source); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

/* ===========================
   Queries methods
   =========================== */

func (q *Queries) GetTagByID(ctx context.Context, id uint64) (dbo.Tag, error) {
	row := q.db.QueryRowContext(ctx, getTagByID, id)
	return parseTagRow(row)
}

func (q *Queries) GetTagByParentAndName(ctx context.Context, parentID *uint64, name string) (dbo.Tag, error) {
	row := q.db.QueryRowContext(ctx, getTagByParentAndName, parentID, name)
	return parseTagRow(row)
}

func (q *Queries) ListTagsByParent(ctx context.Context, parentID *uint64) ([]dbo.Tag, error) {
	rows, err := q.db.QueryContext(ctx, listTagsByParent, parentID)
	if err != nil {
		return nil, err
	}
	return parseTagRows(rows)
}

func (q *Queries) CreateTag(ctx context.Context, t dbo.Tag) error {
	_, err := q.db.ExecContext(ctx, createTag, t.Name, t.ParentID, t.Source)
	return err
}

func (q *Queries) DeleteTag(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteTag, id)
	return err
}

func (q *Queries) QueryTagsByImageID(ctx context.Context, imageID uint64) ([]dbo.Tag, error) {

	rows, err := q.db.QueryContext(ctx, queryTagsByImageID, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags, err := parseTagRows(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func (q *Queries) QueryTags(ctx context.Context) ([]dbo.Tag, error) {

	rows, err := q.db.QueryContext(ctx, queryTags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags, err := parseTagRows(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
func (q *Queries) QueryTagsByACL(ctx context.Context, acl ACLContext) ([]dbo.TagWCount, error) {
	rows, err := q.db.QueryContext(ctx, queryTagsByACL, acl.AsParamArray()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []dbo.TagWCount
	for rows.Next() {
		var t dbo.TagWCount
		if err := rows.Scan(&t.ID, &t.Name, &t.ParentID, &t.Source, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (q *Queries) QueryTagsByParentACLPaged(ctx context.Context, parent uint64, acl ACLContext, from, qty uint64) ([]dbo.TagWCount, error) {
	params := []any{parent}
	params = append(params, acl.AsParamArray()...)
	params = append(params, from)
	params = append(params, qty)
	rows, err := q.db.QueryContext(ctx, queryTagsByParentACLPaged, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []dbo.TagWCount
	for rows.Next() {
		var t dbo.TagWCount
		if err := rows.Scan(&t.ID, &t.Name, &t.ParentID, &t.Source, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (q *Queries) CountTagsByParentACL(ctx context.Context, parent uint64, acl ACLContext) (uint64, error) {
	params := []any{parent}
	params = append(params, acl.AsParamArray()...)
	row := q.db.QueryRowContext(ctx, countTagsByParentACL, params...)
	var count uint64
	err := row.Scan(&count)
	return count, err
}

func (q *Queries) GetTagByIDACL(ctx context.Context, id uint64, acl ACLContext) (dbo.Tag, error) {
	params := []any{id}
	params = append(params, acl.AsParamArray()...)
	row := q.db.QueryRowContext(ctx, getTagByIDACL, params...)
	return parseTagRow(row)
}

//
// =========================================================
// Public API functions
// =========================================================
//

func GetTagByID(db *sql.DB, ctx context.Context, id uint64) (dbo.Tag, error) {
	log.Logger.Debug().Uint64("tag_id", id).Msg("Get tag by ID")

	q := NewQueries(db)
	t, err := q.GetTagByID(ctx, id)
	return t, wrapNotFound(err, "tag")
}

func GetTagByParentAndName(db *sql.DB, ctx context.Context, parentID *uint64, name string) (dbo.Tag, error) {
	logg := logging.Enter(ctx, "dao.tag.get.parent.name", map[string]any{"parent": parentID, "name": name})
	q := NewQueries(db)
	t, err := q.GetTagByParentAndName(ctx, parentID, name)
	return t, returnWrapNotFound(logg, err, "tag")
}

func ListTagsByParent(db *sql.DB, ctx context.Context, parentID *uint64) ([]dbo.Tag, error) {
	if parentID != nil {
		log.Logger.Debug().Uint64("parent_id", *parentID).Msg("List tags by parent")
	} else {
		log.Logger.Debug().Str("parent_id", "NULL").Msg("List tags by parent")
	}

	q := NewQueries(db)
	return q.ListTagsByParent(ctx, parentID)
}

func CreateTag(db *sql.DB, ctx context.Context, t dbo.Tag) (uint64, error) {
	log.Logger.Debug().Object("Tag", logging.WithLevel(zerolog.DebugLevel, &t)).Msg("Create tag")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateTag(ctx, t); err != nil {
		return 0, err
	}
	lastRow, err := q.GetLastId(ctx)

	return lastRow, tx.Commit()
}

func CreateTagTx(q *Queries, ctx context.Context, t dbo.Tag) (uint64, error) {
	log.Logger.Debug().Object("Tag", logging.WithLevel(zerolog.DebugLevel, &t)).Msg("Create tag Tx")
	if err := q.CreateTag(ctx, t); err != nil {
		return 0, err
	}
	return q.GetLastId(ctx)
}

func DeleteTag(db *sql.DB, ctx context.Context, id uint64) error {
	log.Logger.Debug().Uint64("tag_id", id).Msg("Delete tag")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteTag(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}
func InsertTagPath(db *sql.DB, ctx context.Context, path []string) ([]uint64, error) {
	logg := logging.Enter(ctx, "dao.tag.insertPath", map[string]any{"path": path})
	if len(path) == 0 {
		logging.Exit(logg, "empty", nil)
		return nil, nil
	}
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	defer tx.Rollback()
	q := NewQueries(tx)
	var (
		parentID *uint64 = nil
		ids              = make([]uint64, 0, len(path))
	)
	for _, name := range path {
		tag, err := GetTagByParentAndName(db, ctx, parentID, name)
		var id uint64
		switch {
		case err == nil:
			id = *tag.ID
		case !errors.Is(err, ErrDataNotFound):
			logging.ExitErr(logg, err)
			return nil, err
		default:
			tag := dbo.Tag{
				Name:     name,
				ParentID: parentID,
				Source:   "digikam",
			}
			if err := q.CreateTag(ctx, tag); err != nil {
				logging.ExitErr(logg, err)
				return nil, err
			}

			id, err = q.GetLastId(ctx)
			if err != nil {
				logging.ExitErr(logg, err)
				return nil, err
			}
		}
		ids = append(ids, id)
		parentID = &id
	}

	if err := tx.Commit(); err != nil {
		logging.ExitErr(logg, err)
		return nil, err
	}
	logging.Exit(logg, "ok", map[string]any{"ids": ids})
	return ids, nil
}

func QueryTags(db *sql.DB, ctx context.Context) ([]dbo.Tag, error) {
	logg := logging.Enter(ctx, "dao.tag.query", nil)
	q := NewQueries(db)
	tags, err := q.QueryTags(ctx)
	return tags, logging.Return(logg, err)
}

func QueryTagsByACL(db *sql.DB, ctx context.Context, acl ACLContext) ([]dbo.TagWCount, error) {
	logg := logging.Enter(ctx, "dao.tag.query.byACL", map[string]any{"ACL": acl})
	q := NewQueries(db)
	tags, err := q.QueryTagsByACL(ctx, acl)
	return tags, logging.ReturnParams(logg, err, map[string]any{"found": len(tags)})
}
func QueryTagsByParentACLPaged(db *sql.DB, ctx context.Context, parent uint64, acl ACLContext, from, qty uint64) ([]dbo.TagWCount, error) {
	logg := logging.Enter(ctx, "dao.tag.query.byParent.ByACL.paged", map[string]any{"ACL": acl, "parent": parent, "from": from, "qty": qty})
	q := NewQueries(db)
	tags, err := q.QueryTagsByParentACLPaged(ctx, parent, acl, from, qty)
	return tags, logging.ReturnParams(logg, err, map[string]any{"found": len(tags)})
}
func CountTagsByParentACL(db *sql.DB, ctx context.Context, parent uint64, acl ACLContext) (uint64, error) {
	logg := logging.Enter(ctx, "dao.tag.count.byParent.ByACL", map[string]any{"ACL": acl, "parent": parent})
	q := NewQueries(db)
	count, err := q.CountTagsByParentACL(ctx, parent, acl)
	return count, logging.Return(logg, err)
}

func GetTagByIDACL(db *sql.DB, ctx context.Context, tagId uint64, acl ACLContext) (dbo.Tag, error) {
	logg := logging.Enter(ctx, "dao.tag.get.byId.byACL", map[string]any{"ACL": acl})
	q := NewQueries(db)
	tag, err := q.GetTagByIDACL(ctx, tagId, acl)
	return tag, returnWrapNotFound(logg, err, "tag")
}
