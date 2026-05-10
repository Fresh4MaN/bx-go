package repo

import (
	"bitrixgo/bitrixgo/entity"
)

// Repository предоставляет CRUD-операции для сущности типа T.
type Repository[T any] struct {
	client Client
	meta   *entity.Meta
	table  string
}

// NewRepository создаёт репозиторий для сущности T.
func NewRepository[T any](client Client) *Repository[T] {
	meta, err := entity.For[T]()
	if err != nil {
		panic(err)
	}
	return &Repository[T]{
		client: client,
		meta:   meta,
		table:  client.FullTableName(meta.Table),
	}
}

// Meta возвращает метаданные сущности.
func (r *Repository[T]) Meta() *entity.Meta {
	return r.meta
}

// Table возвращает полное имя таблицы с префиксом.
func (r *Repository[T]) Table() string {
	return r.table
}
