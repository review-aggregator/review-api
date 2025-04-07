package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type GoogleReviewAccount struct {
	ID              uuid.UUID `json:"id"`
	PlatformID      uuid.UUID `json:"platform_id"`
	AccountID       string    `json:"account_id"`
	AccountName     string    `json:"account_name"`
	LocationID      string    `json:"location_id"`
	LocationName    string    `json:"location_name"`
	LocationAddress string    `json:"location_address"`
	OAuthToken      string    `json:"oauth_token"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	queryInsertGoogleReviewAccount = `
	INSERT INTO google_review_accounts (platform_id, account_id, account_name, location_id, location_name, location_address, oauth_token)
	VALUES (:platform_id, :account_id, :account_name, :location_id, :location_name, :location_address, :oauth_token)
	`
)

func CreateGoogleReviewAccount(ctx context.Context, account *GoogleReviewAccount) error {
	_, err := db.NamedExecContext(ctx, queryInsertGoogleReviewAccount, account)
	if err != nil {
		log.Error("Error while creating google review account", err)
		return err
	}

	return nil
}
