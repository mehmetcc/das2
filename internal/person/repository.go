package person

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

type PersonRepository interface {
	Create(ctx context.Context, person *Person) error
	GetByID(ctx context.Context, id string) (*Person, error)
	GetByEmail(ctx context.Context, email string) (*Person, error)
	Update(ctx context.Context, person *Person) error
	Delete(ctx context.Context, id string) error
}

type personRepository struct {
	db     *sql.DB
	logger zap.Logger
}

func NewPersonRepository(db *sql.DB, logger zap.Logger) PersonRepository {
	return &personRepository{
		db:     db,
		logger: logger,
	}
}

func (p *personRepository) Create(ctx context.Context, person *Person) error {
	panic("unimplemented")
}

func (p *personRepository) Delete(ctx context.Context, id string) error {
	panic("unimplemented")
}

func (p *personRepository) GetByEmail(ctx context.Context, email string) (*Person, error) {
	panic("unimplemented")
}

func (p *personRepository) GetByID(ctx context.Context, id string) (*Person, error) {
	panic("unimplemented")
}

func (p *personRepository) Update(ctx context.Context, person *Person) error {
	panic("unimplemented")
}
