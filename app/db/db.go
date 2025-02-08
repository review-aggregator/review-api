package db

import (
	"log"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/jmoiron/sqlx"
)

var (
	DB Database
)

type Database struct {
	Sqlx *sqlx.DB
}

func InitDB(dbURL string) error {
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return err
	}

	DB.Sqlx = db
	return nil
}

func GetDBInstance() *Database {
	return &DB
}
