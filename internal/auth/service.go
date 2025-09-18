package auth

import (
	"context"

	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/internal/session"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, email, username, password string) (id.PublicID, error)
	Login(ctx context.Context, email, password string, session session.SessionSummary) (string, error)
}

type authService struct {
	personRepo  person.PersonRepo
	sessionRepo session.SessionRepo
	logger      *zap.Logger
}

func NewAuthenticationService(personRepo person.PersonRepo, sessionRepo session.SessionRepo, logger *zap.Logger) AuthService {
	return &authService{
		personRepo:  personRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
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

func (a *authService) Login(ctx context.Context, email, password string, session session.SessionSummary) (string, error) {
	p, err := a.personRepo.FindByEmail(ctx, email)
	if err != nil {
		a.logger.Error("failed to find person by email", zap.Error(err))
		return "", err
	}

	// compare email & password & activeness
	if !p.IsActive {
		a.logger.Warn("user not active", zap.String("email", email))
		return "", ErrUserNotActive
	}
	if p.Email != email {
		a.logger.Warn("wrong email", zap.String("email", email))
		return "", ErrInvalidCredentials
	}
	if err = bcrypt.CompareHashAndPassword([]byte(p.Password), []byte(password)); err != nil {
		a.logger.Warn("wrong password", zap.String("email", email))
		return "", ErrInvalidCredentials
	}

	session.PersonID = p.ID
	sessionId, err := a.sessionRepo.Create(ctx, session)
	if err != nil {
		return "", err
	}
	a.logger.Debug("created new session", zap.String("sessionId", string(sessionId)))
	return "", nil
}
