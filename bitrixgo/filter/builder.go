package filter

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"

	bxerrors "bitrixgo/bitrixgo/errors"
)

var operators = []string{">=", "<=", ">", "<", "=", "!", "%", "@"}

// Apply добавляет условия фильтра к squirrel SelectBuilder или другому builder с Where.
func Apply(b sq.SelectBuilder, f Filter) (sq.SelectBuilder, error) {
	for key, value := range f {
		cond, err := condition(key, value)
		if err != nil {
			return b, err
		}
		b = b.Where(cond)
	}
	return b, nil
}

// ApplyDelete добавляет условия фильтра к DeleteBuilder.
func ApplyDelete(b sq.DeleteBuilder, f Filter) (sq.DeleteBuilder, error) {
	for key, value := range f {
		cond, err := condition(key, value)
		if err != nil {
			return b, err
		}
		b = b.Where(cond)
	}
	return b, nil
}

func condition(key string, value any) (sq.Sqlizer, error) {
	op, field, err := parseKey(key)
	if err != nil {
		return nil, err
	}

	switch op {
	case "=":
		return sq.Eq{field: value}, nil
	case "!":
		return sq.NotEq{field: value}, nil
	case ">":
		return sq.Gt{field: value}, nil
	case ">=":
		return sq.GtOrEq{field: value}, nil
	case "<":
		return sq.Lt{field: value}, nil
	case "<=":
		return sq.LtOrEq{field: value}, nil
	case "%":
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %% operator requires string value for %s", bxerrors.ErrInvalidFilter, key)
		}
		return sq.Like{field: "%" + s + "%"}, nil
	case "@":
		return inCondition(field, value)
	default:
		return nil, fmt.Errorf("%w: unknown operator in %s", bxerrors.ErrInvalidFilter, key)
	}
}

func inCondition(field string, value any) (sq.Sqlizer, error) {
	switch v := value.(type) {
	case []int:
		if len(v) == 0 {
			return sq.Eq{field: []int{-1}}, nil
		}
		args := make([]any, len(v))
		for i := range v {
			args[i] = v[i]
		}
		return sq.Eq{field: args}, nil
	case []int64:
		if len(v) == 0 {
			return sq.Eq{field: []int64{-1}}, nil
		}
		args := make([]any, len(v))
		for i := range v {
			args[i] = v[i]
		}
		return sq.Eq{field: args}, nil
	case []string:
		if len(v) == 0 {
			return sq.Eq{field: []string{""}}, nil
		}
		args := make([]any, len(v))
		for i := range v {
			args[i] = v[i]
		}
		return sq.Eq{field: args}, nil
	case []any:
		return sq.Eq{field: v}, nil
	default:
		return nil, fmt.Errorf("%w: @ operator requires slice value for %s", bxerrors.ErrInvalidFilter, field)
	}
}

func parseKey(key string) (op string, field string, err error) {
	for _, candidate := range operators {
		if strings.HasPrefix(key, candidate) {
			field = strings.TrimPrefix(key, candidate)
			if field == "" {
				return "", "", fmt.Errorf("%w: missing field name in %s", bxerrors.ErrInvalidFilter, key)
			}
		if candidate == "!" {
			return "!", field, nil
		}
		return candidate, field, nil
		}
	}
	return "", "", fmt.Errorf("%w: invalid filter key %s", bxerrors.ErrInvalidFilter, key)
}

// ParseKey экспортирует разбор ключей фильтра для тестов.
func ParseKey(key string) (op string, field string, err error) {
	return parseKey(key)
}
