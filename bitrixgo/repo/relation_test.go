package repo_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
	"github.com/Fresh4MaN/bx-go/bitrixgo/repo"
)

type M2OContractor struct {
	ID   int64  `bx:"ID;pk;auto"`
	Name string `bx:"UF_NAME"`
}

func (M2OContractor) TableName() string { return "contractor" }

type M2OContract struct {
	ID           int64         `bx:"ID;pk;auto"`
	ContractorID int64         `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
	Contractor   M2OContractor `bx:"rel=Contractor;fk=UF_CONTRACTOR_ID"`
	Name         string        `bx:"UF_NAME"`
}

func (M2OContract) TableName() string { return "contract" }

type ParentContractor struct {
	ID        int64      `bx:"ID;pk;auto"`
	Guid      string     `bx:"UF_GUID;ext"`
	Name      string     `bx:"UF_NAME"`
	Contracts []O2MContract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID;inverse"`
}

func (ParentContractor) TableName() string { return "contractor" }

type O2MContract struct {
	ID           int64  `bx:"ID;pk;auto"`
	ContractorID int64  `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
	Guid         string `bx:"UF_GUID;ext"`
	Name         string `bx:"UF_NAME"`
}

func (O2MContract) TableName() string { return "contract" }

func TestGetListByFKBuildsFilter(t *testing.T) {
	meta, err := entity.For[M2OContract]()
	require.NoError(t, err)

	rel, err := meta.ResolveRelation("Contractor")
	require.NoError(t, err)
	require.Equal(t, "UF_CONTRACTOR_ID", rel.FKColumn)
}

func TestSaveCascadeUpsertByExt(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ParentContractor](client)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .+ FROM b_contractor WHERE UF_GUID").
		WithArgs("c-guid").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "UF_GUID", "UF_NAME"}))
	mock.ExpectExec("INSERT INTO b_contractor").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT .+ FROM b_contract WHERE UF_GUID").
		WithArgs("d-guid").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "UF_CONTRACTOR_ID", "UF_GUID", "UF_NAME"}))
	mock.ExpectExec("INSERT INTO b_contract").
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectCommit()

	item := ParentContractor{
		Guid: "c-guid",
		Name: "Test",
		Contracts: []O2MContract{
			{Guid: "d-guid", Name: "Contract 1"},
		},
	}
	id, err := r.SaveCascade(context.Background(), &item)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncCascadeDeletesOrphans(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ParentContractor](client)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE b_contractor SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE b_contract SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM b_contract WHERE UF_CONTRACTOR_ID = \\? AND ID NOT IN").
		WithArgs(int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	item := ParentContractor{
		ID:   1,
		Guid: "c-guid",
		Name: "Test",
		Contracts: []O2MContract{
			{ID: 10, ContractorID: 1, Guid: "d-guid", Name: "Contract 1"},
		},
	}
	id, err := r.SyncCascade(context.Background(), &item)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSyncCascadeDeletesAllChildrenWhenSliceEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := newTestClient(db, "b_")
	r := repo.NewRepository[ParentContractor](client)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE b_contractor SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM b_contract WHERE UF_CONTRACTOR_ID = \\?").
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	item := ParentContractor{
		ID:        5,
		Guid:      "c-guid",
		Name:      "Test",
		Contracts: nil,
	}
	_, err = r.SyncCascade(context.Background(), &item)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
