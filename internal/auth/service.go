package auth

import (
	"context"

	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/pkg/id"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, email, username, password string) (id.PublicID, error)
}

type authService struct {
	personRepo person.PersonRepo
	logger     *zap.Logger
}

func NewAuthenticationService(personRepo person.PersonRepo, logger *zap.Logger) AuthService {
	return &authService{
		personRepo: personRepo,
		logger:     logger,
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
