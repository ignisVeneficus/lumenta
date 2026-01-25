package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
)

//go:embed schema.sql
var createDatabase string

func (q *Queries) CreateDatabase(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, createDatabase)
	return err
}

const getLastId = `select LAST_INSERT_ID()`

func (q *Queries) GetLastId(ctx context.Context) (uint64, error) {
	row := q.db.QueryRowContext(ctx, getLastId)
	var last_insert_id uint64
	err := row.Scan(&last_insert_id)
	return last_insert_id, err
}

func GetTx(db *sql.DB, ctx context.Context) (*sql.Tx, error) {
	return db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func NewQueries(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}

var ErrDataNotFound = errors.New("data not found")
var ErrDataDuplicateKey = errors.New("duplicate key")

func GetDataNotFoundError(table string) error {
	return fmt.Errorf("%w, table: %s", ErrDataNotFound, table)
}

func NormalizeSQLError(err error) error {
	if err == nil {
		return nil
	}

	// MySQL / MariaDB
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062: // ER_DUP_ENTRY
			return ErrDataDuplicateKey
			/*
				case 1213:
					return ErrDeadlock
				case 1452:
					return ErrForeignKeyViolation
			*/
		}
	}

	return err
}

func CreateDatabase(db *sql.DB, ctx context.Context) error {
	logg := logging.Enter(ctx, "dao.database.create", nil)
	queries := NewQueries(db)
	err := queries.CreateDatabase(ctx)
	if err != nil {
		logging.ExitErr(logg, err)
		return err
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

type ACLContext struct {
	ViewerUserID *uint64
	Role         string
}

func (a ACLContext) IsAnyUser() bool {
	return a.ViewerUserID != nil
}

func (a ACLContext) IsAdmin() bool {
	return a.Role == "admin"
}

func (a ACLContext) AsParamArray() []any {
	return []any{a.IsAnyUser(), a.ViewerUserID, a.IsAdmin()}
}
func (a *ACLContext) MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level) {
	if level <= zerolog.DebugLevel {
		e.Str("role", a.Role)
		logging.Uint64If(e, "userID", a.ViewerUserID)
	}
}

const aclAlbumWhereClause = `
(
  a.acl_scope = 'public'
  OR (a.acl_scope = 'any_user' AND ? = TRUE)
  OR (a.acl_scope = 'user' AND a.acl_user_id = ?)
  OR (a.acl_scope = 'admin' AND ? = TRUE)
)
`
const aclImageWhereClause = `
(
  i.acl_scope = 'public'
  OR (i.acl_scope = 'any_user' AND ? = TRUE)
  OR (i.acl_scope = 'user' AND i.acl_user_id = ?)
  OR (i.acl_scope = 'admin' AND ? = TRUE)
)
`

func wrapNotFound(err error, entity string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return GetDataNotFoundError(entity)
	}
	return err
}
func returnWrapNotFound(logg zerolog.Logger, err error, entity string) error {
	if err == nil {
		logging.Exit(logg, "ok", nil)
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		logging.Exit(logg, "not found", nil)
		return GetDataNotFoundError(entity)
	}
	logging.ExitErr(logg, err)
	return err
}

func buildUint64InClause(ids []uint64) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	return strings.Join(placeholders, ","), args
}
