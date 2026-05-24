package repo

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	bxerrors "github.com/Fresh4MaN/bx-go/bitrixgo/errors"
	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
)

// BatchUpdate содержит поля для обновления одной строки.
type BatchUpdate struct {
	ID     any
	Fields map[string]any
}

// AddMulti вставляет несколько строк пакетными INSERT внутри транзакции.
func (r *Repository[T]) AddMulti(ctx context.Context, items []T) ([]int64, error) {
	if len(items) == 0 {
		return nil, bxerrors.ErrEmptyBatch
	}

	cols := r.meta.InsertColumns()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Column
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	batchSize := r.client.BatchSize()
	var ids []int64

	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]

		b := ps.Insert(r.table).Columns(colNames...)
		for _, item := range chunk {
			values, err := entity.ValuesFromEntity(r.meta, item, cols)
			if err != nil {
				return nil, err
			}
			b = b.Values(values...)
		}

		sqlStr, args, err := b.ToSql()
		if err != nil {
			return nil, fmt.Errorf("repo: build batch insert: %w", err)
		}

		res, err := tx.ExecContext(ctx, sqlStr, args...)
		if err != nil {
			return nil, fmt.Errorf("repo: batch insert: %w", err)
		}

		firstID, err := res.LastInsertId()
		if err != nil {
			rows, _ := res.RowsAffected()
			for i := int64(0); i < rows; i++ {
				ids = append(ids, 0)
			}
			continue
		}
		rowsAffected, _ := res.RowsAffected()
		for i := int64(0); i < rowsAffected; i++ {
			ids = append(ids, firstID+i)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repo: commit: %w", err)
	}
	return ids, nil
}

// UpdateMulti обновляет несколько строк в одной транзакции.
func (r *Repository[T]) UpdateMulti(ctx context.Context, updates []BatchUpdate) error {
	if len(updates) == 0 {
		return bxerrors.ErrEmptyBatch
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, u := range updates {
		if len(u.Fields) == 0 {
			continue
		}
		cols, vals, err := entity.ValuesFromMap(r.meta, u.Fields)
		if err != nil {
			return err
		}
		b := ps.Update(r.table)
		for i, col := range cols {
			b = b.Set(col, vals[i])
		}
		b = b.Where(sq.Eq{r.meta.PrimaryKey: u.ID})

		sqlStr, args, err := b.ToSql()
		if err != nil {
			return fmt.Errorf("repo: build batch update: %w", err)
		}
		if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
			return fmt.Errorf("repo: batch update: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteMulti удаляет строки по первичным ключам в одной транзакции.
func (r *Repository[T]) DeleteMulti(ctx context.Context, ids []any) error {
	if len(ids) == 0 {
		return bxerrors.ErrEmptyBatch
	}

	tx, err := r.client.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	b := ps.Delete(r.table).Where(sq.Eq{r.meta.PrimaryKey: ids})
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build batch delete: %w", err)
	}
	if _, err := tx.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("repo: batch delete: %w", err)
	}

	return tx.Commit()
}
