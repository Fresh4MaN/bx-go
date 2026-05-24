package repo

import (
	"context"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"

	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
	bxerrors "github.com/Fresh4MaN/bx-go/bitrixgo/errors"
)

// Upsert обновляет по PK или ищет по ext → update/insert.
func (r *Repository[T]) Upsert(ctx context.Context, item *T) (int64, error) {
	return r.upsert(ctx, r.client.DB(), item)
}

// GetByExt возвращает строку по внешнему идентификатору.
func (r *Repository[T]) GetByExt(ctx context.Context, extValue any) (*T, error) {
	if r.meta.ExtColumn == "" {
		return nil, fmt.Errorf("repo: entity %s has no ext column", r.meta.Type.Name())
	}
	items, err := r.queryByExt(ctx, r.client.DB(), extValue, 2)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, bxerrors.ErrNotFound
	}
	if len(items) > 1 {
		return nil, bxerrors.ErrDuplicateExt
	}
	return &items[0], nil
}

func (r *Repository[T]) upsert(ctx context.Context, q querier, item *T) (int64, error) {
	row := reflectValueOf(item)

	isZero, err := entity.IsZeroPrimaryKey(r.meta, row)
	if err != nil {
		return 0, err
	}
	if !isZero {
		pk, err := entity.PrimaryKeyValueOf(r.meta, row)
		if err != nil {
			return 0, err
		}
		if err := r.updateEntity(ctx, q, pk, item); err != nil {
			return 0, err
		}
		return pkAsInt64(pk), nil
	}

	extVal, hasExt, err := entity.ExtValueOf(r.meta, row)
	if err != nil {
		return 0, err
	}
	if hasExt && !entity.IsEmptyExt(extVal) {
		existing, err := r.queryByExt(ctx, q, extVal, 2)
		if err != nil {
			return 0, err
		}
		if len(existing) > 1 {
			return 0, bxerrors.ErrDuplicateExt
		}
		if len(existing) == 1 {
			pk, err := entity.PrimaryKeyValueOf(r.meta, reflect.ValueOf(&existing[0]).Elem())
			if err != nil {
				return 0, err
			}
			if err := r.updateEntity(ctx, q, pk, item); err != nil {
				return 0, err
			}
			if err := entity.SetPrimaryKey(r.meta, row, pkAsInt64(pk)); err != nil {
				return 0, err
			}
			return pkAsInt64(pk), nil
		}
	}

	id, err := r.add(ctx, q, item)
	if err != nil {
		return 0, err
	}
	if err := entity.SetPrimaryKey(r.meta, row, id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository[T]) updateEntity(ctx context.Context, q querier, id any, item *T) error {
	cols := r.meta.UpdateColumns()
	fields := make(map[string]any, len(cols))
	rv := reflectValueOf(item)
	for _, f := range cols {
		v, err := entity.FieldToDB(rv.Field(f.Index), f)
		if err != nil {
			return err
		}
		fields[f.Column] = v
	}
	return r.upsertUpdate(ctx, q, id, fields)
}

func (r *Repository[T]) queryByExt(ctx context.Context, q querier, extValue any, limit uint64) ([]T, error) {
	b := ps.Select(r.meta.SelectColumns()...).
		From(r.table).
		Where(sq.Eq{r.meta.ExtColumn: extValue}).
		Limit(limit)

	sqlStr, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("repo: build select by ext: %w", err)
	}
	return r.queryRows(ctx, q, sqlStr, args...)
}

func upsertReflect(ctx context.Context, q querier, client Client, meta *entity.Meta, value reflect.Value) (int64, error) {
	isZero, err := entity.IsZeroPrimaryKey(meta, value)
	if err != nil {
		return 0, err
	}
	if !isZero {
		pk, err := entity.PrimaryKeyValueOf(meta, value)
		if err != nil {
			return 0, err
		}
		if err := updateReflect(ctx, q, client, meta, pk, value); err != nil {
			return 0, err
		}
		return pkAsInt64(pk), nil
	}

	extVal, hasExt, err := entity.ExtValueOf(meta, value)
	if err != nil {
		return 0, err
	}
	if hasExt && !entity.IsEmptyExt(extVal) {
		table := client.FullTableName(meta.Table)
		b := ps.Select(meta.SelectColumns()...).
			From(table).
			Where(sq.Eq{meta.ExtColumn: extVal}).
			Limit(2)
		sqlStr, args, err := b.ToSql()
		if err != nil {
			return 0, fmt.Errorf("repo: build select by ext: %w", err)
		}
		rows, err := q.QueryContext(ctx, sqlStr, args...)
		if err != nil {
			return 0, fmt.Errorf("repo: query by ext: %w", err)
		}
		defer rows.Close()

		var found []reflect.Value
		for rows.Next() {
			itemPtr := reflect.New(meta.Type)
			if err := entity.ScanRow(rows, meta, itemPtr.Interface()); err != nil {
				return 0, err
			}
			found = append(found, itemPtr.Elem())
		}
		if err := rows.Err(); err != nil {
			return 0, err
		}
		if len(found) > 1 {
			return 0, bxerrors.ErrDuplicateExt
		}
		if len(found) == 1 {
			pk, err := entity.PrimaryKeyValueOf(meta, found[0])
			if err != nil {
				return 0, err
			}
			if err := updateReflect(ctx, q, client, meta, pk, value); err != nil {
				return 0, err
			}
			id := pkAsInt64(pk)
			if err := entity.SetPrimaryKey(meta, value, id); err != nil {
				return 0, err
			}
			return id, nil
		}
	}

	id, err := insertReflect(ctx, q, client, meta, value)
	if err != nil {
		return 0, err
	}
	if err := entity.SetPrimaryKey(meta, value, id); err != nil {
		return 0, err
	}
	return id, nil
}

func updateReflect(ctx context.Context, q querier, client Client, meta *entity.Meta, id any, value reflect.Value) error {
	table := client.FullTableName(meta.Table)
	cols := meta.UpdateColumns()
	b := ps.Update(table)
	for _, f := range cols {
		v, err := entity.FieldToDB(value.Field(f.Index), f)
		if err != nil {
			return err
		}
		b = b.Set(f.Column, v)
	}
	b = b.Where(sq.Eq{meta.PrimaryKey: id})
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build update: %w", err)
	}
	res, err := q.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("repo: update: %w", err)
	}
	_ = res
	return nil
}

func pkAsInt64(pk any) int64 {
	switch v := pk.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		return reflect.ValueOf(pk).Int()
	}
}
