package repo_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bxerrors "bitrixgo/bitrixgo/errors"
	"bitrixgo/bitrixgo/repo"
)

type ExtItem struct {
	ID   int64  `bx:"ID;pk;auto"`
	Guid string `bx:"UF_GUID;ext"`
	Name string `bx:"UF_NAME"`
}

func (ExtItem) TableName() string { return "ext_item" }

type testClient struct {
	db     *sql.DB
	prefix string
}

func (c testClient) DB() *sql.DB { return c.db }

func (c testClient) FullTableName(logical string) string {
	return c.prefix + logical
}

func (c testClient) BatchSize() int { return 100 }

func newTestClient(db *sql.DB, prefix string) testClient {
	return testClient{db: db, prefix: prefix}
}

func TestUpsertInsertWithoutExt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectExec("INSERT INTO b_ext_item").
		WillReturnResult(sqlmock.NewResult(5, 1))

	item := ExtItem{Name: "test"}
	id, err := r.Upsert(context.Background(), &item)
	require.NoError(t, err)
	assert.Equal(t, int64(5), id)
	assert.Equal(t, int64(5), item.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertUpdateByPK(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectExec("UPDATE b_ext_item SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	item := ExtItem{ID: 3, Name: "updated"}
	id, err := r.Upsert(context.Background(), &item)
	require.NoError(t, err)
	assert.Equal(t, int64(3), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertFindByExtAndUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	cols := []string{"ID", "UF_GUID", "UF_NAME"}
	selectRows := sqlmock.NewRows(cols).AddRow(int64(7), "guid-1", "old")

	mock.ExpectQuery("SELECT .+ FROM b_ext_item WHERE UF_GUID").
		WithArgs("guid-1").
		WillReturnRows(selectRows)
	mock.ExpectExec("UPDATE b_ext_item SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	item := ExtItem{Guid: "guid-1", Name: "new"}
	id, err := r.Upsert(context.Background(), &item)
	require.NoError(t, err)
	assert.Equal(t, int64(7), id)
	assert.Equal(t, int64(7), item.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByExtNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectQuery("SELECT .+ FROM b_ext_item WHERE UF_GUID").
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "UF_GUID", "UF_NAME"}))

	_, err = r.GetByExt(context.Background(), "missing")
	require.ErrorIs(t, err, bxerrors.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByExtDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	cols := []string{"ID", "UF_GUID", "UF_NAME"}
	selectRows := sqlmock.NewRows(cols).
		AddRow(int64(1), "dup", "a").
		AddRow(int64(2), "dup", "b")

	mock.ExpectQuery("SELECT .+ FROM b_ext_item WHERE UF_GUID").
		WithArgs("dup").
		WillReturnRows(selectRows)

	_, err = r.GetByExt(context.Background(), "dup")
	require.ErrorIs(t, err, bxerrors.ErrDuplicateExt)
	require.NoError(t, mock.ExpectationsWereMet())
}
