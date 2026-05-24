package repo_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	bxerrors "github.com/Fresh4MaN/bx-go/bitrixgo/errors"
	"github.com/Fresh4MaN/bx-go/bitrixgo/repo"
)

func TestUpdateNoChangesStillSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectExec("UPDATE b_ext_item SET").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT 1 FROM b_ext_item WHERE ID").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	err = r.Update(context.Background(), int64(7), map[string]any{"UF_NAME": "same"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMissingRecordReturnsNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectExec("UPDATE b_ext_item SET").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT 1 FROM b_ext_item WHERE ID").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	err = r.Update(context.Background(), int64(999), map[string]any{"UF_NAME": "new"})
	require.ErrorIs(t, err, bxerrors.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWithChangesSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ExtItem](client)

	mock.ExpectExec("UPDATE b_ext_item SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = r.Update(context.Background(), int64(7), map[string]any{"UF_NAME": "changed"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
