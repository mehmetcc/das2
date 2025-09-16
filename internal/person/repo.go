package person

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
)

type PersonDTO struct {
	Email    string
	Username string
	Password string
}

type PersonRepo interface {
	Create(ctx context.Context, dto *PersonDTO) (id.PublicID, error)
}

type personRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewPersonRepo(db *sql.DB, logger *zap.Logger) PersonRepo {
	return &personRepo{
		db:     db,
		logger: logger,
	}
}

const (
	insertPersonQuery = `
						INSERT INTO persons (email, username, password, role, is_active, is_deleted)
						VALUES ($1, $2, $3, $4, $5, $6)
						RETURNING id, public_id, created_at, updated_at
						`
)

/**
 * Go is usually written by people who claims to hate boilerplate, Java code.
 * Those people hate Hibernate, Spring, any crap Microsoft comes up with for C# etc.
 * And then they write 50 lines of boilerplate to insert a single record into a database.
 * The only way to access a database is either a raw SQL query, or an ORM that is just like Hibernate, but worse.
 * How this ecosystem is even worth the hype amk?
 **/
func (p *personRepo) Create(ctx context.Context, dto *PersonDTO) (id.PublicID, error) {
	// I must admit, this wasn't working until I rewrote everything with Claude
	row := p.db.QueryRowContext(ctx,
		insertPersonQuery,
		strings.ToLower(strings.TrimSpace(dto.Email)),
		strings.TrimSpace(dto.Username),
		dto.Password,
		RoleUser,
		false,
		false,
	)

	var publicID id.PublicID
	var id int64
	var createdAt, updatedAt time.Time

	if err := row.Scan(&id, &publicID, &createdAt, &updatedAt); err != nil {
		// context canceled/deadline
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			p.logger.Warn("create person canceled/timed out", zap.Error(err))
			return "", err
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				switch pgErr.ConstraintName {
				case "persons_email_key":
					p.logger.Debug("duplicate email", zap.String("email", dto.Email))
					return "", ErrDuplicateEmail
				case "persons_username_key":
					p.logger.Debug("duplicate username", zap.String("username", dto.Username))
					return "", ErrDuplicateUsername
				default:
					// Handle unique index on expression (lower(email)) by inspecting detail
					det := strings.ToLower(pgErr.Detail)
					if strings.Contains(det, "lower(email)") || strings.Contains(det, "(email)") {
						p.logger.Debug("duplicate email (detail match)", zap.String("email", dto.Email))
						return "", ErrDuplicateEmail
					}
					if strings.Contains(det, "(username)") {
						p.logger.Debug("duplicate username (detail match)", zap.String("username", dto.Username))
						return "", ErrDuplicateUsername
					}
				}
			}
			p.logger.Error("postgres error",
				zap.String("code", string(pgErr.Code)),
				zap.String("msg", pgErr.Message),
				zap.String("detail", pgErr.Detail),
			)
			return "", err
		}

		// Fallback: match by message text if driver wrapped the error
		// lol :D
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "persons_email_key") || strings.Contains(msg, "lower(email)") {
			p.logger.Debug("duplicate email (fallback)", zap.String("email", dto.Email))
			return "", ErrDuplicateEmail
		}
		if strings.Contains(msg, "persons_username_key") || strings.Contains(msg, "(username)") {
			p.logger.Debug("duplicate username (fallback)", zap.String("username", dto.Username))
			return "", ErrDuplicateUsername
		}

		p.logger.Error("driver/scan error", zap.Error(err))
		return "", err
	}

	p.logger.Debug("person created",
		zap.Int64("id", id),
		zap.String("public_id", string(publicID)),
	)

	return publicID, nil
}
