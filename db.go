package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var db *sql.DB
var appSecret string

func initDB() {
	err := godotenv.Load()
	if err != nil {
		panic("failed to load .env file")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbUser,
		dbPassword,
		dbName,
		dbHost,
		dbPort,
		dbSSLMode,
	)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	appSecret = os.Getenv("APP_SECRET")

	if appSecret == "" {
		panic("APP_SECRET is required")
	}
}
