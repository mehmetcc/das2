package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mehmetcc/das2/internal/config"
)

func Init(cfg *config.Config) (*sql.DB, error) {
	dsn := cfg.DbConfig.DSN
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DbConfig.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DbConfig.MaxConnLifetime)

	return db, nil
}
