package services

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleReviewConfig struct {
	CLIENTID     string `mapstructure:"client_id"`
	CLIENTSECRET string `mapstructure:"client_secret"`
}

var oauthConfig *oauth2.Config

func InitGoogleReview(googleClientID, googleClientSecret string) {
	oauthConfig = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/business.manage"},
		Endpoint:     google.Endpoint,
	}
}
