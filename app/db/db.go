package db

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	DB Database
)

type Database struct {
	Sqlx *sqlx.DB
}

func InitDB(dbURL string) error {
	fmt.Println("dburl: ", dbURL)
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
