package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Init() (*sql.DB, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}
