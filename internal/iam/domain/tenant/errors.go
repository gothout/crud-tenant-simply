package tenant

import "errors"

var (
	ErrDocumentDuplicated = errors.New("document already exists")
	ErrNotFound           = errors.New("tenant not found")
	ErrInvalidInput       = errors.New("invalid input data")
)
