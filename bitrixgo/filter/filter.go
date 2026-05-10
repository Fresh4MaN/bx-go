package filter

// Filter — map фильтров в стиле Bitrix. Ключи с префиксами операторов: =FIELD, !FIELD, >FIELD и т.д.
type Filter map[string]any
