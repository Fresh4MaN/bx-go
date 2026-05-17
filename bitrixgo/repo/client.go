package repo

import (
	"context"
	"database/sql"
)

// Client предоставляет доступ к БД для репозиториев.
type Client interface {
	DB() *sql.DB
	FullTableName(logical string) string
	BatchSize() int
}

type tableClient struct {
	Client
}

func (c tableClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.DB().QueryContext(ctx, query, args...)
}

