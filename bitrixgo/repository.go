package bitrixgo

import "bitrixgo/bitrixgo/repo"

// NewRepository создаёт типизированный репозиторий для сущности T.
func NewRepository[T any](client *Client) *repo.Repository[T] {
	return repo.NewRepository[T](client)
}
