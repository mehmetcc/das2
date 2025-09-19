package auth

import (
	"context"

	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/internal/session"
	"github.com/mehmetcc/das2/internal/token"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, email, username, password string) (id.PublicID, error)
	Login(ctx context.Context, email, password string, sess session.SessionSummary) (*LoginResult, error)
	Refresh(ctx context.Context, refreshToken, userAgent, ip, deviceID string) (*LoginResult, error)
}

type authService struct {
	personRepo   person.PersonRepo
	sessionRepo  session.SessionRepo
	tokenService token.TokenService
	logger       *zap.Logger
}

func NewAuthenticationService(personRepo person.PersonRepo, sessionRepo session.SessionRepo, tokenService token.TokenService, logger *zap.Logger) AuthService {
	return &authService{
		personRepo:   personRepo,
		sessionRepo:  sessionRepo,
		tokenService: tokenService,
		logger:       logger,
	}
}

func (a *authService) Register(ctx context.Context, email, username, password string) (id.PublicID, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.logger.Error("failed to hash password", zap.Error(err))
		return "", err
	}

	id, err := a.personRepo.Create(ctx, &person.PersonDTO{
		Email:    email,
		Username: username,
		Password: string(hashed),
	})
	if err != nil {
		return "", err
	}

	return id, nil
}

type LoginResult struct {
	AccessToken      string
	AccessExpiresAt  int64 // unix seconds
	RefreshToken     string
	RefreshExpiresAt int64 // unix seconds
	Session          session.SessionSummary
}

func (a *authService) Login(ctx context.Context, email, password string, sess session.SessionSummary) (*LoginResult, error) {
	p, err := a.personRepo.FindByEmail(ctx, email)
	if err != nil {
		a.logger.Error("failed to find person by email", zap.Error(err))
		return nil, err
	}

	// compare email & password & activeness
	if !p.IsActive {
		a.logger.Warn("user not active", zap.String("email", email))
		return nil, ErrUserNotActive
	}
	if p.Email != email {
		a.logger.Warn("wrong email", zap.String("email", email))
		return nil, ErrInvalidCredentials
	}
	if err = bcrypt.CompareHashAndPassword([]byte(p.Password), []byte(password)); err != nil {
		a.logger.Warn("wrong password", zap.String("email", email))
		return nil, ErrInvalidCredentials
	}

	sess.PersonID = p.ID
	sessionId, err := a.sessionRepo.Create(ctx, sess)
	if err != nil {
		return nil, err
	}
	a.logger.Debug("created new session", zap.String("sessionId", string(sessionId)))
	sess.ID = string(sessionId)

	issued, err := a.tokenService.Issue(ctx, p, sessionId, token.IssueMeta{
		UserAgent: sess.UserAgent,
		IP:        sess.LastUsedIP,
		DeviceID:  sess.DeviceID,
	})
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:      issued.AccessToken,
		AccessExpiresAt:  issued.AccessExpiresAt.Unix(),
		RefreshToken:     issued.RefreshToken,
		RefreshExpiresAt: issued.RefreshExpiresAt.Unix(),
		Session:          sess,
	}, nil
}

func (a *authService) Refresh(ctx context.Context, refreshToken, userAgent, ip, deviceID string) (*LoginResult, error) {
	issued, err := a.tokenService.Refresh(ctx, refreshToken, token.IssueMeta{
		UserAgent: userAgent,
		IP:        ip,
		DeviceID:  deviceID,
	})
	if err != nil {
		return nil, err
	}
	// best-effort: we don't know session here; TouchLastUsed is skipped unless we thread session id via token repo lookup return.
	return &LoginResult{
		AccessToken:      issued.AccessToken,
		AccessExpiresAt:  issued.AccessExpiresAt.Unix(),
		RefreshToken:     issued.RefreshToken,
		RefreshExpiresAt: issued.RefreshExpiresAt.Unix(),
	}, nil
}
