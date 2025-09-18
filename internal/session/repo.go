package session

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

type SessionRepo interface {
	Create(ctx context.Context, summary SessionSummary) error
	Delete(ctx context.Context, id string) error
}

const (
	createSessionQuery = `
		INSERT INTO session_summaries (
			device_id, device_name, platform, created_at, last_used_ip, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	deleteSessionQuery = `
		DELETE FROM session_summaries WHERE id = $1
	`
)

type sessionRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewSessionRepo(db *sql.DB, logger *zap.Logger) SessionRepo {
	return &sessionRepo{
		db:     db,
		logger: logger,
	}
}

func (s *sessionRepo) Create(ctx context.Context, summary SessionSummary) error {
	_, err := s.db.ExecContext(ctx, createSessionQuery,
		summary.DeviceID,
		summary.DeviceName,
		summary.Platform,
		summary.CreatedAt,
		summary.LastUsedIP,
		summary.UserAgent,
	)
	if err != nil {
		s.logger.Error("failed to create session", zap.Error(err))
	}
	return err
}

func (s *sessionRepo) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, deleteSessionQuery, id)
	if err != nil {
		s.logger.Error("failed to delete session", zap.String("id", id), zap.Error(err))
	}
	return err
}
