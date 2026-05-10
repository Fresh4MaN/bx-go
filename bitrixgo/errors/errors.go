package errors

import "errors"

var (
	ErrNotFound      = errors.New("bitrixgo: record not found")
	ErrDuplicateKey  = errors.New("bitrixgo: duplicate key")
	ErrEmptyBatch    = errors.New("bitrixgo: empty batch")
	ErrInvalidFilter = errors.New("bitrixgo: invalid filter key")
)
