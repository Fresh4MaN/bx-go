package filter_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Fresh4MaN/bx-go/bitrixgo/filter"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		key      string
		op       string
		field    string
		wantFail bool
	}{
		{"=ID", "=", "ID", false},
		{"!STATUS", "!", "STATUS", false},
		{">PRICE", ">", "PRICE", false},
		{">=PRICE", ">=", "PRICE", false},
		{"<=PRICE", "<=", "PRICE", false},
		{"<PRICE", "<", "PRICE", false},
		{"%NAME", "%", "NAME", false},
		{"@ID", "@", "ID", false},
		{"INVALID", "", "", true},
	}

	for _, tt := range tests {
		op, field, err := filter.ParseKey(tt.key)
		if tt.wantFail {
			require.Error(t, err, tt.key)
			continue
		}
		require.NoError(t, err, tt.key)
		assert.Equal(t, tt.op, op, tt.key)
		assert.Equal(t, tt.field, field, tt.key)
	}
}

func TestApplyEquals(t *testing.T) {
	ps := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	b := ps.Select("*").From("t")
	b, err := filter.Apply(b, filter.Filter{"=ID": 1})
	require.NoError(t, err)

	sql, args, err := b.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "ID = ?")
	assert.Equal(t, []any{1}, args)
}

func TestApplyLike(t *testing.T) {
	ps := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	b := ps.Select("*").From("t")
	b, err := filter.Apply(b, filter.Filter{"%NAME": "test"})
	require.NoError(t, err)

	sql, args, err := b.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "NAME LIKE ?")
	assert.Equal(t, []any{"%test%"}, args)
}

func TestApplyIn(t *testing.T) {
	ps := sq.StatementBuilder.PlaceholderFormat(sq.Question)
	b := ps.Select("*").From("t")
	b, err := filter.Apply(b, filter.Filter{"@ID": []int64{1, 2, 3}})
	require.NoError(t, err)

	sql, args, err := b.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "ID IN")
	assert.Len(t, args, 3)
}
