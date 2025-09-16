package database

import (
	"context"
	"database/sql"

	"github.com/mehmetcc/das2/migrations"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return err
	}
	return nil
}

func SetMigrationLogger(logger *zap.Logger) {
	goose.SetLogger(gooseZapLogger{s: logger.Sugar()})
}

type gooseZapLogger struct{ s *zap.SugaredLogger }

func (l gooseZapLogger) Printf(format string, v ...interface{}) {
	l.s.Infof(format, v...)
}

func (l gooseZapLogger) Fatalf(format string, v ...interface{}) {
	l.s.Errorf(format, v...)
}
