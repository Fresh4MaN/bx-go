package entity_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
)

type mockTableClient struct {
	db *sql.DB
}

func (c mockTableClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

func (c mockTableClient) FullTableName(logical string) string {
	return logical
}

func TestPreloadInverse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	parentMeta, err := entity.For[ParentContractor]()
	require.NoError(t, err)
	childMeta, err := entity.For[Contract]()
	require.NoError(t, err)

	childCols := childMeta.SelectColumns()
	rows := sqlmock.NewRows(childCols).
		AddRow(int64(10), int64(1), "Договор 1", time.Time{}).
		AddRow(int64(11), int64(1), "Договор 2", time.Time{}).
		AddRow(int64(12), int64(2), "Договор 3", time.Time{})

	mock.ExpectQuery("SELECT .+ FROM a_boostrade_lkk_contract WHERE UF_CONTRACTOR_ID IN").
		WithArgs(int64(1), int64(2)).
		WillReturnRows(rows)

	parents := []ParentContractor{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
	}

	client := mockTableClient{db: db}
	err = entity.Preload(context.Background(), client, parentMeta, parents, []string{"Contracts"})
	require.NoError(t, err)

	require.Len(t, parents[0].Contracts, 2)
	assert.Equal(t, "Договор 1", parents[0].Contracts[0].Name)
	assert.Equal(t, int64(1), parents[0].Contracts[0].ContractorID)
	require.Len(t, parents[1].Contracts, 1)
	assert.Equal(t, "Договор 3", parents[1].Contracts[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExtValueOf(t *testing.T) {
	type Item struct {
		ID   int64  `bx:"ID;pk;auto;table=t"`
		Guid string `bx:"UF_GUID;ext"`
	}
	meta, err := entity.For[Item]()
	require.NoError(t, err)

	rv := reflect.ValueOf(Item{Guid: "abc"})
	val, hasExt, err := entity.ExtValueOf(meta, rv)
	require.NoError(t, err)
	assert.True(t, hasExt)
	assert.Equal(t, "abc", val)
	assert.False(t, entity.IsEmptyExt(val))
	assert.True(t, entity.IsEmptyExt(""))
}
