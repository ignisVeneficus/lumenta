package dao

import (
	"context"
	"database/sql"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const tagFields = `t.id, t.name, t.parent_id, t.source`

const getTagByID = `SELECT ` + tagFields + ` FROM tags t WHERE t.id = ?`

const getTagByParentAndName = `SELECT ` + tagFields + `FROM tags t
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

func (q *Queries) QuertyTagsByImageID(ctx context.Context, imageID uint64) ([]dbo.Tag, error) {

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

func GetTagByID(db *sql.DB, ctx context.Context, id uint64) (dbo.Tag, error) {
	log.Logger.Debug().Uint64("tag_id", id).Msg("Get tag by ID")

	q := NewQueries(db)
	t, err := q.GetTagByID(ctx, id)
	return t, wrapNotFound(err, "tag")
}

func GetTagByParentAndName(db *sql.DB, ctx context.Context, parentID *uint64, name string) (dbo.Tag, error) {
	if parentID != nil {
		log.Logger.Debug().Uint64("parent_id", *parentID).Str("name", name).Msg("Get tag by parent and name")
	} else {
		log.Logger.Debug().Str("parent_id", "NULL").Str("name", name).Msg("Get tag by parent and name")
	}
	q := NewQueries(db)
	t, err := q.GetTagByParentAndName(ctx, parentID, name)
	return t, wrapNotFound(err, "tag")
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

	log.Logger.Debug().Strs("path", path).Msg("Insert tag path")

	if len(path) == 0 {
		return nil, nil
	}

	tx, err := GetTx(db, ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	var (
		parentID *uint64
		ids      = make([]uint64, 0, len(path))
	)

	for _, name := range path {
		tag := dbo.Tag{
			Name:     name,
			ParentID: parentID,
			Source:   "digikam",
		}

		if err := q.CreateTag(ctx, tag); err != nil {
			return nil, err
		}

		id, err := q.GetLastId(ctx)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
		parentID = &id
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ids, nil
}
