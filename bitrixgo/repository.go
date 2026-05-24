package bitrixgo

import (
	"context"

	"github.com/Fresh4MaN/bx-go/bitrixgo/repo"
)

// NewRepository создаёт типизированный репозиторий для сущности T.
func NewRepository[T any](client *Client) *repo.Repository[T] {
	return repo.NewRepository[T](client)
}

// DeleteCascade удаляет дочерние записи и родителя в одной транзакции.
func DeleteCascade[Parent any, Child any](ctx context.Context, client *Client, parentID any) error {
	return repo.DeleteCascade[Parent, Child](ctx, client, parentID)
}
