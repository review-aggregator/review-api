package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
)

const (
	queryInsertUser = `
	INSERT INTO users(user_id, email, password, first_name, last_name, company_id, created_at)
	VALUES(:user_id, :email, :password, :first_name, :last_name, :company_id, NOW())`

	queryGetUserByID = `
	SELECT
		u.user_id,
		u.first_name,
		u.last_name,
		u.email,
		u.password,
		u.company_id
	FROM
		users u
	WHERE
		u.user_id = :user_id`

	queryGetUserByEmail = `
		SELECT
			u.user_id,
			u.first_name,
			u.last_name,
			u.email,
			u.password
		FROM
			users u
		WHERE
			email = :email`

	queryUpdateUserByID = `
		UPDATE users SET %s
		WHERE user_id = :user_id
		RETURNING *`
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email" validate:"required,email"`
	Password  string    `db:"password" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func CreateUser(ctx context.Context, user *User) error {
	_, err := db.NamedExecContext(ctx, queryInsertUser, user)
	if err != nil {
		log.Error("Error while creating user", err)
		return err
	}

	return nil
}

func GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User

	err := db.NamedGetContext(ctx, &user, queryGetUserByID, map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No user found for id: ", userID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching user by user_id", err)
		return nil, err
	}

	return &user, nil
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	err := db.NamedGetContext(ctx, &user, queryGetUserByEmail, map[string]interface{}{
		"email": email,
	})
	if err != nil {
		log.Info("No user found for email: ", email)
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching user by email: ", err)
		return nil, err
	}

	return &user, nil
}

func UpdateUser(ctx context.Context, user *User) error {
	userMap := structs.Map(user)
	query := fmt.Sprintf(queryUpdateUserByID, db.GetFormattedColumnNames(db.GetStringMapKeys(userMap), "user_id"))

	err := db.NamedExecContextReturnObj(ctx, query, userMap, user)
	if err != nil {
		log.Error("Error while updating user", err)
		return err
	}

	return nil
}
