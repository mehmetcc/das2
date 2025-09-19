package token

import (
	"context"
	"database/sql"
	"time"

	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
)

// RefreshTokenDTO carries the necessary fields to persist a refresh token record.
type RefreshTokenDTO struct {
	PersonID  int64
	SessionID id.SessionID
	TokenHash string
	ExpiresAt time.Time
	UserAgent string
	IP        string
	DeviceID  string
}

type RefreshTokenRepo interface {
	Create(ctx context.Context, dto RefreshTokenDTO) (string, error)
	RevokeByID(ctx context.Context, id string) error
	FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (*RefreshTokenLookup, error)
	RotateCreateNext(ctx context.Context, oldID string, dto RefreshTokenDTO) (string, error)
	MarkReuseAndRevokeChain(ctx context.Context, id string) error
}

const (
	insertRefreshTokenQuery = `
						INSERT INTO refresh_tokens (
						person_id, session_id, token_hash, expires_at, user_agent, ip, device_id
						) VALUES ($1, $2, $3, $4, $5, $6, $7)
						RETURNING id
						`
	revokeRefreshTokenByIDQuery = `
						UPDATE refresh_tokens
						SET revoked_at = COALESCE(revoked_at, now())
						WHERE id = $1 AND revoked_at IS NULL
						`
	findActiveByHashQuery = `
						SELECT rt.id, rt.person_id, rt.session_id, rt.expires_at, p.public_id, p.role
						FROM refresh_tokens rt
						JOIN persons p ON p.id = rt.person_id
						WHERE rt.token_hash = $1
						  AND rt.revoked_at IS NULL
						  AND rt.rotated_at IS NULL
						  AND rt.expires_at > $2
						LIMIT 1
						`
)

type refreshTokenRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewRefreshTokenRepo(db *sql.DB, logger *zap.Logger) RefreshTokenRepo {
	return &refreshTokenRepo{db: db, logger: logger}
}

func (r *refreshTokenRepo) Create(ctx context.Context, dto RefreshTokenDTO) (string, error) {
	var rid string
	row := r.db.QueryRowContext(ctx, insertRefreshTokenQuery,
		dto.PersonID,
		string(dto.SessionID),
		dto.TokenHash,
		dto.ExpiresAt,
		dto.UserAgent,
		dto.IP,
		dto.DeviceID,
	)
	if err := row.Scan(&rid); err != nil {
		r.logger.Error("failed to insert refresh token", zap.Error(err))
		return "", err
	}
	return rid, nil
}

func (r *refreshTokenRepo) RevokeByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, revokeRefreshTokenByIDQuery, id)
	if err != nil {
		r.logger.Error("failed to revoke refresh token", zap.String("id", id), zap.Error(err))
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// no-op if already revoked or not found
		r.logger.Debug("no refresh token revoked (not found or already revoked)", zap.String("id", id))
	}
	return nil
}

// RefreshTokenLookup includes data needed for rotation and access issuance.
type RefreshTokenLookup struct {
	ID             string
	PersonID       int64
	SessionID      id.SessionID
	ExpiresAt      time.Time
	PersonPublicID id.PublicID
	PersonRole     person.Role
}

func (r *refreshTokenRepo) FindActiveByHash(ctx context.Context, tokenHash string, now time.Time) (*RefreshTokenLookup, error) {
	row := r.db.QueryRowContext(ctx, findActiveByHashQuery, tokenHash, now)
	var rec RefreshTokenLookup
	if err := row.Scan(&rec.ID, &rec.PersonID, &rec.SessionID, &rec.ExpiresAt, &rec.PersonPublicID, &rec.PersonRole); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("failed to lookup refresh token by hash", zap.Error(err))
		return nil, err
	}
	return &rec, nil
}

// RotateCreateNext performs rotation: inserts a new token row and marks the old as rotated linking replaced_by.
func (r *refreshTokenRepo) RotateCreateNext(ctx context.Context, oldID string, dto RefreshTokenDTO) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var newID string
	err = tx.QueryRowContext(ctx, insertRefreshTokenQuery,
		dto.PersonID,
		string(dto.SessionID),
		dto.TokenHash,
		dto.ExpiresAt,
		dto.UserAgent,
		dto.IP,
		dto.DeviceID,
	).Scan(&newID)
	if err != nil {
		r.logger.Error("failed to insert rotated refresh token", zap.Error(err))
		return "", err
	}

	_, err = tx.ExecContext(ctx, `UPDATE refresh_tokens SET rotated_at = now(), replaced_by = $2 WHERE id = $1 AND rotated_at IS NULL AND revoked_at IS NULL`, oldID, newID)
	if err != nil {
		r.logger.Error("failed to mark refresh token as rotated", zap.Error(err))
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return newID, nil
}

// MarkReuseAndRevokeChain revokes the token and its rotation chain (defensive response to reuse attacks).
func (r *refreshTokenRepo) MarkReuseAndRevokeChain(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, replaced_by FROM refresh_tokens WHERE id = $1
			UNION ALL
			SELECT rt.id, rt.replaced_by FROM refresh_tokens rt JOIN chain c ON rt.id = c.replaced_by
		)
		UPDATE refresh_tokens rt
		SET revoked_at = COALESCE(rt.revoked_at, now())
		FROM chain c
		WHERE rt.id = c.id
	`, id)
	if err != nil {
		r.logger.Error("failed to revoke refresh chain", zap.String("id", id), zap.Error(err))
	}
	return err
}
