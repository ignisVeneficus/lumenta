package dao

import (
	"context"
	"errors"

	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/db/dbo"

	"database/sql"

	"github.com/rs/zerolog/log"
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

func (q *Queries) GetUserById(ctx context.Context, id uint64) (dbo.User, error) {
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

func (q *Queries) DeleteUser(ctx context.Context, id uint64) error {
	_, err := q.db.ExecContext(ctx, deleteUser, id)
	return err
}

func (q *Queries) UpdateUserPassword(ctx context.Context, userID uint64, passHash string) error {
	_, err := q.db.ExecContext(ctx, updateUserPassword, passHash, userID)
	return err
}

func GetUserById(db *sql.DB, ctx context.Context, id uint64) (dbo.User, error) {
	log.Logger.Debug().Uint64("user_id", id).Msg("Get User")
	q := NewQueries(db)
	u, err := q.GetUserById(ctx, id)
	return u, wrapNotFound(err, "user")
}
func AuthenticateUser(db *sql.DB, ctx context.Context, username string, password string) (dbo.User, error) {

	log.Logger.Debug().Str("username", username).Msg("Authenticate User")

	q := NewQueries(db)
	ur, hash, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return dbo.User{}, wrapNotFound(err, "user")
	}

	if !auth.VerifyPassword(hash, password) {
		return dbo.User{}, GetDataNotFoundError("user")
	}

	return ur, nil
}
func CreateUser(db *sql.DB, ctx context.Context, u dbo.User, passHash string) error {
	log.Logger.Debug().Str("username", u.Username).Msg("Create User")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.CreateUser(ctx, u, passHash); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteUser(db *sql.DB, ctx context.Context, id uint64) error {
	log.Logger.Debug().Uint64("user_id", id).Msg("Delete User")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.DeleteUser(ctx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func UpdateUser(db *sql.DB, ctx context.Context, u dbo.User) error {
	log.Logger.Debug().Uint64("user", *u.ID).Msg("Update User")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)
	if err := q.UpdateUser(ctx, u); err != nil {
		return err
	}
	return tx.Commit()
}

func ChangePasswordByUser(db *sql.DB, ctx context.Context, username string, currentPassword string, newPassword string) error {
	log.Logger.Debug().Str("username", username).Msg("User initiated password change")
	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := NewQueries(tx)

	u, hash, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetDataNotFoundError("users")
		}
		return err
	}

	if !auth.VerifyPassword(hash, currentPassword) {
		return GetDataNotFoundError("users")
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := q.UpdateUserPassword(ctx, *u.ID, newHash); err != nil {
		return err
	}
	return tx.Commit()
}
func ResetPasswordByAdmin(db *sql.DB, ctx context.Context, username string, newPassword string) error {

	log.Logger.Debug().
		Str("target_user", username).
		Msg("Admin initiated password reset")

	tx, err := GetTx(db, ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := NewQueries(tx)

	u, _, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetDataNotFoundError("users")
		}
		return err
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := q.UpdateUserPassword(ctx, *u.ID, newHash); err != nil {
		return err
	}

	return tx.Commit()
}
