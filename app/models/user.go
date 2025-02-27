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
		INSERT INTO users(id, clerk_id, name, email, created_at, updated_at)
		VALUES(:id, :clerk_id, :name, :email, NOW(), NOW())`

	queryGetUserByID = `
		SELECT u.id, u.name, u.email, u.created_at
		FROM users u
		WHERE u.id = :id`

	queryGetUserByClerkID = `
		SELECT u.id, u.clerk_id, u.name, u.email, u.created_at
		FROM users u
		WHERE u.clerk_id = :clerk_id`

	queryGetUserByEmail = `
		SELECT u.id, u.name, u.email, u.created_at
		FROM users u
		WHERE email = :email`

	queryUpdateUserByID = `
		UPDATE users SET %s
		WHERE id = :id
		RETURNING *`
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	ClerkID   string    `db:"clerk_id" json:"clerk_id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email" validate:"required,email"`
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

func GetUserByUserID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User

	err := db.NamedGetContext(ctx, &user, queryGetUserByID, map[string]interface{}{
		"id": userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No user found for id: ", userID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching user by id", err)
		return nil, err
	}

	return &user, nil
}

func GetUserByClerkID(ctx context.Context, clerkID string) (*User, error) {
	var user User

	err := db.NamedGetContext(ctx, &user, queryGetUserByClerkID, map[string]interface{}{
		"clerk_id": clerkID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No user found for clerk id: ", clerkID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching user by clerk id", err)
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
	query := fmt.Sprintf(queryUpdateUserByID, db.GetFormattedColumnNames(db.GetStringMapKeys(userMap), "id"))

	err := db.NamedExecContextReturnObj(ctx, query, userMap, user)
	if err != nil {
		log.Error("Error while updating user", err)
		return err
	}

	return nil
}
