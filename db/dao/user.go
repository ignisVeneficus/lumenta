package dao

import (
	"context"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	"database/sql"
)

const userFields = `u.id, u.username, u.email, u.role, u.disabled, u.created_at, u.pass_hash`
const getUserById = `SELECT ` + userFields + ` FROM users u WHERE u.id = ?`
const getUserByUsername = `SELECT ` + userFields + ` FROM users u WHERE u.username = ? AND u.disabled = FALSE`
const createUser = `INSERT INTO users (username, pass_hash, email, role, disabled) VALUES (?,?,?,?,?)`
const updateUser = `UPDATE users SET email=?, role=?, disabled=? WHERE id=?`
const deleteUser = `DELETE FROM users WHERE id = ?`

const updateUserPassword = `UPDATE users SET pass_hash = ? WHERE id = ?`

func parseUser(row *sql.Row) (dbo.User, string, error) {
	var hash string
	var u dbo.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Disabled, &u.CreatedAt, &hash)
	return u, hash, err
}

func (q *Queries) GetUserById(ctx context.Context, id dbo.UserID) (dbo.User, error) {
	row := q.db.QueryRowContext(ctx, getUserById, id)
	dbo, _, err := parseUser(row)
	return dbo, err
}

func (q *Queries) CreateUser(ctx context.Context, u dbo.User, passHash string) error {
	_, err := q.db.ExecContext(ctx, createUser, u.Username, passHash, u.Email, u.Role, u.Disabled)
	return err
}
func (q *Queries) GetUserByUsername(ctx context.Context, username string) (dbo.User, string, error) {
	row := q.db.QueryRowContext(ctx, getUserByUsername, username)
	return parseUser(row)
}
func (q *Queries) UpdateUser(ctx context.Context, u dbo.User) error {
	_, err := q.db.ExecContext(ctx, updateUser, u.Email, u.Role, u.Disabled, u.ID)
	return err
}

func (q *Queries) DeleteUser(ctx context.Context, id dbo.UserID) error {
	_, err := q.db.ExecContext(ctx, deleteUser, id)
	return err
}

func (q *Queries) UpdateUserPassword(ctx context.Context, userID dbo.UserID, passHash string) error {
	_, err := q.db.ExecContext(ctx, updateUserPassword, passHash, userID)
	return err
}

func GetUserById(db *sql.DB, c context.Context, id dbo.UserID) (dbo.User, error) {
	logScope, ctx := logging.Enter(c, "dao/user/get/byId", id, map[string]any{"user_id": id})
	q := NewQueries(db)
	u, err := q.GetUserById(ctx, id)
	return u, returnWrapNotFound(logScope, err, "user")
}
func AuthenticateUser(db *sql.DB, c context.Context, username string, password string) (dbo.User, error) {
	logScope, ctx := logging.Enter(c, "dao/user/get/authenticate", username, map[string]any{"name": username})

	q := NewQueries(db)
	ur, hash, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return dbo.User{}, returnWrapNotFound(logScope, err, "user")
	}

	if !auth.VerifyPassword(hash, password) {
		ur = dbo.User{}
		err = GetDataNotFoundError("user")
	}

	return ur, logging.Return(logScope, err)
}
func CreateUser(db *sql.DB, c context.Context, u dbo.User, passHash string) error {
	logScope, ctx := logging.Enter(c, "dao/user/create", u.Username, map[string]any{"user": u})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateUser(ctx, u, passHash); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}

func DeleteUser(db *sql.DB, c context.Context, id dbo.UserID) error {
	logScope, ctx := logging.Enter(c, "dao/user/delete", id, map[string]any{"user_id": id})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteUser(ctx, id); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

func UpdateUser(db *sql.DB, c context.Context, u dbo.User) error {
	logScope, ctx := logging.Enter(c, "dao/user/update", u.ID, map[string]any{"user": u})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateUser(ctx, u); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}

func ChangePasswordByUser(db *sql.DB, c context.Context, username string, currentPassword string, newPassword string) error {
	logScope, ctx := logging.Enter(c, "dao/user/update/password", username, map[string]any{"username": username})
	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	u, hash, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return returnWrapNotFound(logScope, err, "user")
	}

	if !auth.VerifyPassword(hash, currentPassword) {
		err := GetDataNotFoundError("users")
		logging.ExitErr(logScope, err)
		return err
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	if err := q.UpdateUserPassword(ctx, *u.ID, newHash); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	return logging.Return(logScope, tx.Commit())
}
func ResetPasswordByAdmin(db *sql.DB, c context.Context, username string, newPassword string) error {
	logScope, ctx := logging.Enter(c, "dao/user/update/password/admin", username, map[string]any{"username": username})

	tx, err := GetTx(db, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	u, _, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return returnWrapNotFound(logScope, err, "users")
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	if err := q.UpdateUserPassword(ctx, *u.ID, newHash); err != nil {
		logging.ExitErr(logScope, err)
		return err
	}

	return logging.Return(logScope, tx.Commit())
}
