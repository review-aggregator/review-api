package models

import (
	"context"

	"github.com/google/uuid"
)

type GoogleReviewToken struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
}

const (
	queryInsertGoogleToken = `
	INSERT INTO google_review_tokens (user_id, token)
	VALUES (:user_id, :token)
	`
)

func CreateGoogleToken(ctx context.Context, token *GoogleReviewToken) error {
	_, err := db.NamedExecContext(ctx, queryInsertGoogleToken, token)
	if err != nil {
		log.Error("Error while creating google token", err)
		return err
	}

	return nil
}
