package person

import (
	"time"

	"github.com/mehmetcc/das2/pkg/id"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// TODO: add Salt in the future
type Person struct {
	ID        int64       `json:"id" db:"id"`
	PublicID  id.PublicID `json:"public_id" db:"public_id"`
	Email     string      `json:"email" db:"email"`
	Username  string      `json:"username" db:"username"`
	Password  string      `json:"-" db:"password"`
	Role      Role        `json:"role" db:"role"`
	IsActive  bool        `json:"is_active" db:"is_active"`
	IsDeleted bool        `json:"is_deleted" db:"is_deleted"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

func NewPerson(id int64, publicID id.PublicID, email, username, password string) *Person {
	return &Person{
		ID:        id,
		PublicID:  publicID,
		Email:     email,
		Username:  username,
		Password:  password, // store a hash here
		Role:      RoleUser,
		IsActive:  false, // will be activated via email
		IsDeleted: false,
	}
}
