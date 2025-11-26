package user

import "errors"

var (
	ErrEmailDuplicated    = errors.New("email already exists")
	ErrNotFound           = errors.New("user not found")
	ErrInvalidInput       = errors.New("invalid input data")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
