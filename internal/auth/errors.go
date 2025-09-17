package auth

import "errors"

var (
	ErrWrongPassword = errors.New("wrong password")
	ErrWrongUsername = errors.New("wrong username")
	ErrWrongEmail    = errors.New("wrong email")
	ErrUserNotActive = errors.New("user not active")
)
