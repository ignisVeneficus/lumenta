package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
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

func CreateDatabase(db *sql.DB, c context.Context) error {
	logScope, ctx := logging.Enter(c, "dao.database.create", nil, nil)
	queries := NewQueries(db)
	err := queries.CreateDatabase(ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

const aclWhereClauseGuest = ` %sacl_level = 0 `

const aclWhereClauseUser = ` %sacl_level <= 1 
AND %sacl_user_id in (0,?)
`
const aclWhereClauseAdmin = ` 1 = 1 `

func CreateAclWhere(alias string, acl dbo.ACLContext) (string, []any) {
	if alias != "" {
		alias += "."
	}
	params := make([]any, 0)
	switch acl.Role {
	case dbo.RoleAdmin:
		return aclWhereClauseAdmin, params
	case dbo.RoleUser:
		return fmt.Sprintf(aclWhereClauseUser, alias, alias), append(params, acl.ViewerUserID)
	//case dbo.RoleGuest
	default:
		return fmt.Sprintf(aclWhereClauseGuest, alias), params
	}
}

func wrapNotFound(err error, entity string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return GetDataNotFoundError(entity)
	}
	return err
}
func returnWrapNotFound(scope logging.LogScope, err error, entity string) error {
	if err == nil {
		logging.Exit(scope, "ok", nil)
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		logging.Exit(scope, "not found", nil)
		return GetDataNotFoundError(entity)
	}
	logging.ExitErr(scope, err)
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
func Placeholder(qty int) string {
	if qty < 1 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", qty), ",")
}
