package token

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/pkg/id"
)

type Claims struct {
	Sub  id.PublicID  `json:"sub"`
	SID  id.SessionID `json:"sid"`
	Role person.Role  `json:"role"`
	jwt.RegisteredClaims
}
