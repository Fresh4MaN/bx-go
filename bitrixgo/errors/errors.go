package errors

import "errors"

var (
	ErrNotFound         = errors.New("bitrixgo: record not found")
	ErrDuplicateKey     = errors.New("bitrixgo: duplicate key")
	ErrDuplicateExt     = errors.New("bitrixgo: duplicate ext")
	ErrEmptyBatch       = errors.New("bitrixgo: empty batch")
	ErrInvalidFilter    = errors.New("bitrixgo: invalid filter key")
	ErrRelationNotFound = errors.New("bitrixgo: relation not found")
	ErrInvalidRelation  = errors.New("bitrixgo: invalid relation")
)
