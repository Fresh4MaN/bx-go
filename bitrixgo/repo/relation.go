package repo

import (
	"context"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"

	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
	"github.com/Fresh4MaN/bx-go/bitrixgo/filter"
	"github.com/Fresh4MaN/bx-go/bitrixgo/query"
)

// GetListByFK возвращает строки дочерней таблицы по FK родителя.
func (r *Repository[T]) GetListByFK(ctx context.Context, relation string, parentID any, opts query.Options) ([]T, error) {
	rel, err := r.meta.ResolveRelation(relation)
	if err != nil {
		return nil, err
	}
	if opts.Filter == nil {
		opts.Filter = filter.Filter{}
	}
	opts.Filter["="+rel.FKColumn] = parentID
	return r.GetList(ctx, opts)
}

// AddCascade вставляет родителя (если PK=0) и дочернюю запись в одной транзакции.
func (r *Repository[T]) AddCascade(ctx context.Context, item *T) (int64, error) {
	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := reflectValueOf(item)
	for i := range r.meta.Relations {
		rel := &r.meta.Relations[i]
		if rel.Inverse {
			continue
		}
		nested := entity.NestedValue(row, rel)
		if nested.Kind() == reflect.Pointer {
			if nested.IsNil() {
				continue
			}
			nested = nested.Elem()
		}
		if nested.IsZero() {
			continue
		}

		parentMeta, err := entity.ForType(rel.Target)
		if err != nil {
			return 0, err
		}

		parentID, err := upsertReflect(ctx, tx, r.client, parentMeta, nested)
		if err != nil {
			return 0, err
		}
		if err := entity.SetFK(r.meta, rel, row, parentID); err != nil {
			return 0, err
		}
	}

	id, err := r.upsert(ctx, tx, item)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("repo: commit: %w", err)
	}
	return id, nil
}

// SaveCascade сохраняет родителя и дочерние one-to-many записи (upsert, без delete лишних).
func (r *Repository[T]) SaveCascade(ctx context.Context, item *T) (int64, error) {
	return r.cascadeSave(ctx, item, false)
}

// SyncCascade сохраняет родителя и дочерние one-to-many записи (upsert по PK/ext)
// и удаляет дочерние строки с тем же FK, которых нет во входном slice.
func (r *Repository[T]) SyncCascade(ctx context.Context, item *T) (int64, error) {
	return r.cascadeSave(ctx, item, true)
}

func (r *Repository[T]) cascadeSave(ctx context.Context, item *T, syncDelete bool) (int64, error) {
	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := reflectValueOf(item)

	parentID, err := r.upsert(ctx, tx, item)
	if err != nil {
		return 0, err
	}

	for i := range r.meta.Relations {
		rel := &r.meta.Relations[i]
		if !rel.Inverse {
			continue
		}

		childMeta, err := entity.ForType(rel.Target)
		if err != nil {
			return 0, err
		}
		fkRel := entity.ChildFKRelation(rel.FKColumn, rel.FKIndex)

		var keptIDs []any
		sliceVal := row.Field(rel.FieldIndex)
		for j := 0; j < sliceVal.Len(); j++ {
			child := sliceVal.Index(j)
			if err := entity.SetFK(childMeta, fkRel, child, parentID); err != nil {
				return 0, err
			}
			childID, err := upsertReflect(ctx, tx, r.client, childMeta, child)
			if err != nil {
				return 0, err
			}
			keptIDs = append(keptIDs, childID)
		}

		if syncDelete {
			if err := deleteInverseOrphans(ctx, tx, r.client, childMeta, rel.FKColumn, parentID, keptIDs); err != nil {
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("repo: commit: %w", err)
	}
	return parentID, nil
}

func deleteInverseOrphans(ctx context.Context, q querier, client Client, childMeta *entity.Meta, fkColumn string, parentID any, keepIDs []any) error {
	table := client.FullTableName(childMeta.Table)
	b := ps.Delete(table).Where(sq.Eq{fkColumn: parentID})
	if len(keepIDs) > 0 {
		b = b.Where(sq.NotEq{childMeta.PrimaryKey: keepIDs})
	}
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build sync delete children: %w", err)
	}
	if _, err := q.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("repo: sync delete children: %w", err)
	}
	return nil
}

// DeleteCascade удаляет дочерние записи и родителя в одной транзакции.
func DeleteCascade[Parent any, Child any](ctx context.Context, client Client, parentID any) error {
	parentRepo := NewRepository[Parent](client)
	childMeta, err := entity.For[Child]()
	if err != nil {
		return err
	}
	rel, err := entity.RelationTo[Parent](childMeta)
	if err != nil {
		return err
	}

	childRepo := NewRepository[Child](client)

	tx, err := client.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	delChild := ps.Delete(childRepo.table).Where(sq.Eq{rel.FKColumn: parentID})
	sqlStr, args, err := delChild.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build cascade delete child: %w", err)
	}
	if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("repo: cascade delete child: %w", err)
	}

	if err := parentRepo.delete(ctx, tx, parentID); err != nil {
		return err
	}

	return tx.Commit()
}

func insertReflect(ctx context.Context, q querier, client Client, meta *entity.Meta, value reflect.Value) (int64, error) {
	table := client.FullTableName(meta.Table)
	cols := meta.InsertColumns()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Column
	}

	ptr := reflect.New(meta.Type)
	ptr.Elem().Set(value)
	values, err := entity.ValuesFromEntity(meta, ptr.Interface(), cols)
	if err != nil {
		return 0, err
	}

	b := ps.Insert(table).Columns(colNames...).Values(values...)
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return 0, fmt.Errorf("repo: build insert: %w", err)
	}
	res, err := q.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("repo: insert: %w", err)
	}
	return res.LastInsertId()
}
