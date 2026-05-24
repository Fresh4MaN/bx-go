package query

import "github.com/Fresh4MaN/bx-go/bitrixgo/filter"

// Options управляет параметрами GetList (в стиле Bitrix getList).
type Options struct {
	Select []string
	Filter filter.Filter
	Order  map[string]string
	Limit  uint64
	Offset uint64
	With   []string
}
