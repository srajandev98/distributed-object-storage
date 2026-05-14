package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBConnString string
	AppSecret    string
	HTTPAddr     string
	StorageNodes []string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	appSecret := os.Getenv("APP_SECRET")
	if appSecret == "" {
		return Config{}, fmt.Errorf("APP_SECRET is required")
	}

	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbUser,
		dbPassword,
		dbName,
		dbHost,
		dbPort,
		dbSSLMode,
	)

	return Config{
		DBConnString: connStr,
		AppSecret:    appSecret,
		HTTPAddr:     ":8080",
		StorageNodes: []string{
			"storage/node1",
			"storage/node2",
			"storage/node3",
		},
	}, nil
}
