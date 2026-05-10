package repo

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"bitrixgo/bitrixgo/entity"
	bxerrors "bitrixgo/bitrixgo/errors"
	"bitrixgo/bitrixgo/filter"
	"bitrixgo/bitrixgo/query"
)

var ps = sq.StatementBuilder.PlaceholderFormat(sq.Question)

// GetByID возвращает одну строку по первичному ключу.
func (r *Repository[T]) GetByID(ctx context.Context, id any) (*T, error) {
	opts := query.Options{
		Filter: filter.Filter{"=" + r.meta.PrimaryKey: id},
		Limit:  1,
	}
	rows, err := r.GetList(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, bxerrors.ErrNotFound
	}
	return &rows[0], nil
}

// GetList возвращает строки по параметрам запроса.
func (r *Repository[T]) GetList(ctx context.Context, opts query.Options) ([]T, error) {
	cols := opts.Select
	if len(cols) == 0 {
		cols = r.meta.SelectColumns()
	}

	b := ps.Select(cols...).From(r.table)
	var err error
	if len(opts.Filter) > 0 {
		b, err = filter.Apply(b, opts.Filter)
		if err != nil {
			return nil, err
		}
	}
	for col, dir := range opts.Order {
		b = b.OrderBy(fmt.Sprintf("%s %s", col, dir))
	}
	if opts.Limit > 0 {
		b = b.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		b = b.Offset(opts.Offset)
	}

	sqlStr, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("repo: build select: %w", err)
	}

	dbRows, err := r.client.DB().QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("repo: query: %w", err)
	}
	defer dbRows.Close()

	var result []T
	for dbRows.Next() {
		var item T
		if err := entity.ScanRow(dbRows, r.meta, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, dbRows.Err()
}

// Add вставляет новую строку и возвращает сгенерированный PK при auto-increment.
func (r *Repository[T]) Add(ctx context.Context, item *T) (int64, error) {
	cols := r.meta.InsertColumns()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Column
	}
	values, err := entity.ValuesFromEntity(r.meta, item, cols)
	if err != nil {
		return 0, err
	}

	b := ps.Insert(r.table).Columns(colNames...).Values(values...)
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return 0, fmt.Errorf("repo: build insert: %w", err)
	}

	res, err := r.client.DB().ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("repo: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, nil
	}
	return id, nil
}

// Update обновляет поля строки по первичному ключу.
func (r *Repository[T]) Update(ctx context.Context, id any, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	cols, vals, err := entity.ValuesFromMap(r.meta, fields)
	if err != nil {
		return err
	}

	b := ps.Update(r.table)
	for i, col := range cols {
		b = b.Set(col, vals[i])
	}
	b = b.Where(sq.Eq{r.meta.PrimaryKey: id})

	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build update: %w", err)
	}

	res, err := r.client.DB().ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("repo: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return bxerrors.ErrNotFound
	}
	return nil
}

// Delete удаляет строку по первичному ключу.
func (r *Repository[T]) Delete(ctx context.Context, id any) error {
	b := ps.Delete(r.table).Where(sq.Eq{r.meta.PrimaryKey: id})
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("repo: build delete: %w", err)
	}
	res, err := r.client.DB().ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("repo: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return bxerrors.ErrNotFound
	}
	return nil
}

// Count возвращает число строк, удовлетворяющих фильтру.
func (r *Repository[T]) Count(ctx context.Context, f filter.Filter) (int64, error) {
	b := ps.Select("COUNT(*)").From(r.table)
	var err error
	if len(f) > 0 {
		b, err = filter.Apply(b, f)
		if err != nil {
			return 0, err
		}
	}
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return 0, err
	}
	var n int64
	err = r.client.DB().QueryRowContext(ctx, sqlStr, args...).Scan(&n)
	return n, err
}
