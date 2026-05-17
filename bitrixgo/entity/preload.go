package entity

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	bxerrors "bitrixgo/bitrixgo/errors"
)

// Querier выполняет SQL-запросы ( *sql.DB или *sql.Tx ).
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// TableClient добавляет имя таблицы с префиксом.
type TableClient interface {
	Querier
	FullTableName(logical string) string
}

// Preload подгружает many-to-one связи отдельными SELECT.
func Preload(ctx context.Context, client TableClient, meta *Meta, rows any, with []string) error {
	if len(with) == 0 {
		return nil
	}

	rv := reflect.ValueOf(rows)
	if rv.Kind() != reflect.Slice {
		return fmt.Errorf("entity: preload: expected slice, got %s", rv.Kind())
	}
	if rv.Len() == 0 {
		return nil
	}

	for _, name := range with {
		rel, err := meta.RelationByName(name)
		if err != nil {
			return err
		}
		if err := preloadRelation(ctx, client, meta, rv, rel); err != nil {
			return err
		}
	}
	return nil
}

func preloadRelation(ctx context.Context, client TableClient, childMeta *Meta, rows reflect.Value, rel *Relation) error {
	parentMeta, err := ForType(rel.Target)
	if err != nil {
		return err
	}

	ids := collectFKValues(rows, rel.FKIndex)
	if len(ids) == 0 {
		return nil
	}

	parents, err := loadByIDs(ctx, client, parentMeta, ids)
	if err != nil {
		return err
	}

	parentMap, err := indexByPrimaryKey(parentMeta, parents)
	if err != nil {
		return err
	}

	for i := 0; i < rows.Len(); i++ {
		row := rows.Index(i)
		fk := row.Field(rel.FKIndex)
		if fk.Kind() == reflect.Pointer {
			if fk.IsNil() {
				continue
			}
			fk = fk.Elem()
		}
		key, err := normalizeKey(fk.Interface())
		if err != nil {
			continue
		}
		parent, ok := parentMap[key]
		if !ok {
			continue
		}
		row.Field(rel.FieldIndex).Set(parent)
	}
	return nil
}

func collectFKValues(rows reflect.Value, fkIndex int) []any {
	seen := make(map[string]struct{})
	var ids []any
	for i := 0; i < rows.Len(); i++ {
		fk := rows.Index(i).Field(fkIndex)
		if fk.Kind() == reflect.Pointer {
			if fk.IsNil() {
				continue
			}
			fk = fk.Elem()
		}
		if fk.IsZero() {
			continue
		}
		key, err := normalizeKey(fk.Interface())
		if err != nil {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ids = append(ids, fk.Interface())
	}
	return ids
}

func loadByIDs(ctx context.Context, client TableClient, meta *Meta, ids []any) (reflect.Value, error) {
	table := client.FullTableName(meta.Table)
	cols := meta.SelectColumns()
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s IN (%s)",
		strings.Join(cols, ", "),
		table,
		meta.PrimaryKey,
		placeholders,
	)

	rows, err := client.QueryContext(ctx, query, ids...)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("entity: preload query: %w", err)
	}
	defer rows.Close()

	sliceType := reflect.SliceOf(meta.Type)
	result := reflect.MakeSlice(sliceType, 0, len(ids))
	for rows.Next() {
		ptr := reflect.New(meta.Type)
		if err := ScanRow(rows, meta, ptr.Interface()); err != nil {
			return reflect.Value{}, err
		}
		result = reflect.Append(result, ptr.Elem())
	}
	return result, rows.Err()
}

func indexByPrimaryKey(meta *Meta, parents reflect.Value) (map[string]reflect.Value, error) {
	pkField, ok := meta.FieldByColumn(meta.PrimaryKey)
	if !ok {
		return nil, fmt.Errorf("%w: primary key %s", bxerrors.ErrInvalidRelation, meta.PrimaryKey)
	}

	out := make(map[string]reflect.Value, parents.Len())
	for i := 0; i < parents.Len(); i++ {
		parent := parents.Index(i)
		pk := parent.Field(pkField.Index)
		key, err := normalizeKey(pk.Interface())
		if err != nil {
			return nil, err
		}
		out[key] = parent
	}
	return out, nil
}

func normalizeKey(v any) (string, error) {
	switch x := v.(type) {
	case int64:
		return fmt.Sprintf("%d", x), nil
	case int:
		return fmt.Sprintf("%d", x), nil
	case int32:
		return fmt.Sprintf("%d", x), nil
	case uint64:
		return fmt.Sprintf("%d", x), nil
	case string:
		return x, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// PrimaryKeyValueOf извлекает значение первичного ключа из сущности.
func PrimaryKeyValueOf(meta *Meta, rv reflect.Value) (any, error) {
	pkField, ok := meta.FieldByColumn(meta.PrimaryKey)
	if !ok {
		return nil, fmt.Errorf("entity: primary key %s not found", meta.PrimaryKey)
	}
	fv := rv.Field(pkField.Index)
	return fieldValueForRead(fv)
}

func fieldValueForRead(fv reflect.Value) (any, error) {
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return int64(0), nil
		}
		fv = fv.Elem()
	}
	return fv.Interface(), nil
}

// SetFK записывает значение внешнего ключа в поле структуры.
func SetFK(meta *Meta, rel *Relation, row reflect.Value, parentID any) error {
	fk := row.Field(rel.FKIndex)
	return assignValue(fk, parentID)
}

func assignValue(fv reflect.Value, value any) error {
	if !fv.IsValid() {
		return nil
	}
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		fv = fv.Elem()
	}
	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)
		return nil
	}
	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))
		return nil
	}
	return fmt.Errorf("entity: cannot assign %T to %s", value, fv.Type())
}

// NestedValue возвращает значение вложенной rel-структуры.
func NestedValue(row reflect.Value, rel *Relation) reflect.Value {
	return row.Field(rel.FieldIndex)
}

// IsZeroPrimaryKey проверяет, что первичный ключ не задан (нулевой).
func IsZeroPrimaryKey(meta *Meta, rv reflect.Value) (bool, error) {
	pkField, ok := meta.FieldByColumn(meta.PrimaryKey)
	if !ok {
		return false, fmt.Errorf("entity: primary key %s not found", meta.PrimaryKey)
	}
	fv := rv.Field(pkField.Index)
	if fv.Kind() == reflect.Pointer {
		return fv.IsNil() || fv.Elem().IsZero(), nil
	}
	return fv.IsZero(), nil
}

// SetPrimaryKey устанавливает первичный ключ после INSERT.
func SetPrimaryKey(meta *Meta, rv reflect.Value, id int64) error {
	pkField, ok := meta.FieldByColumn(meta.PrimaryKey)
	if !ok {
		return nil
	}
	return assignValue(rv.Field(pkField.Index), id)
}
