package query

import "bitrixgo/bitrixgo/filter"

// Options управляет параметрами GetList (в стиле Bitrix getList).
type Options struct {
	Select []string
	Filter filter.Filter
	Order  map[string]string
	Limit  uint64
	Offset uint64
}
