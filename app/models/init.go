package models

import (
	database "github.com/review-aggregator/review-api/app/db"
	"github.com/review-aggregator/review-api/app/utils"
)

var (
	log = utils.CreateLogger()
	db  = database.GetDBInstance()
)
