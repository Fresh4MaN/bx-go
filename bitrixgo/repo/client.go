package repo

import "database/sql"

// Client предоставляет доступ к БД для репозиториев.
type Client interface {
	DB() *sql.DB
	FullTableName(logical string) string
	BatchSize() int
}
