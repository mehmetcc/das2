// internal/session/repo.go
package session

import (
	"context"
	"database/sql"

	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
)

type SessionRepo interface {
	Create(ctx context.Context, summary SessionSummary) (id.SessionID, error)
	Delete(ctx context.Context, id string) error
}

const (
	createSessionQuery = `
						INSERT INTO sessions (
						person_id, device_id, device_name, platform, created_at, last_used_ip, user_agent
						) VALUES ($1, $2, $3, $4, COALESCE($5, now()), $6, $7)
						RETURNING id
						`
	deleteSessionQuery = `
						DELETE FROM sessions WHERE id = $1
						`
)

type sessionRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewSessionRepo(db *sql.DB, logger *zap.Logger) SessionRepo {
	return &sessionRepo{db: db, logger: logger}
}

func (s *sessionRepo) Create(ctx context.Context, summary SessionSummary) (id.SessionID, error) {
	var sid string
	err := s.db.QueryRowContext(ctx, createSessionQuery,
		summary.PersonID,
		summary.DeviceID,
		summary.DeviceName,
		summary.Platform,
		summary.CreatedAt,
		summary.LastUsedIP,
		summary.UserAgent,
	).Scan(&sid)
	if err != nil {
		s.logger.Error("failed to create session", zap.Error(err))
		return "", err
	}
	return id.SessionID(sid), nil
}

func (s *sessionRepo) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, deleteSessionQuery, id)
	if err != nil {
		s.logger.Error("failed to delete session", zap.String("id", id), zap.Error(err))
	}
	return err
}
