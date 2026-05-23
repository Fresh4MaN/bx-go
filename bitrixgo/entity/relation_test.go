package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitrixgo/bitrixgo/entity"
)

type Contractor struct {
	ID   int64  `bx:"ID;pk;auto"`
	Name string `bx:"UF_NAME"`
	Inn  string `bx:"UF_INN"`
}

func (Contractor) TableName() string { return "a_boostrade_lkk_contractor" }

type Contract struct {
	ID           int64      `bx:"ID;pk;auto"`
	ContractorID int64      `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
	Contractor   Contractor `bx:"rel=Contractor;fk=UF_CONTRACTOR_ID"`
	Name         string     `bx:"UF_NAME"`
	DateFrom     time.Time  `bx:"UF_DATE_FROM"`
}

func (Contract) TableName() string { return "a_boostrade_lkk_contract" }

func TestRelationMeta(t *testing.T) {
	meta, err := entity.For[Contract]()
	require.NoError(t, err)

	require.Len(t, meta.Relations, 1)
	assert.Equal(t, "Contractor", meta.Relations[0].Name)
	assert.Equal(t, "UF_CONTRACTOR_ID", meta.Relations[0].FKColumn)

	cols := meta.SelectColumns()
	assert.Contains(t, cols, "UF_CONTRACTOR_ID")
	assert.Contains(t, cols, "UF_NAME")
	assert.NotContains(t, cols, "Contractor")

	insertCols := meta.InsertColumns()
	for _, c := range insertCols {
		assert.NotEqual(t, "Contractor", c.Name)
	}

	rel, err := meta.RelationByName("Contractor")
	require.NoError(t, err)
	assert.Equal(t, "UF_CONTRACTOR_ID", rel.FKColumn)

	rel, err = meta.ResolveRelation("")
	require.NoError(t, err)
	assert.Equal(t, "Contractor", rel.Name)
}

func TestRelationTo(t *testing.T) {
	meta, err := entity.For[Contract]()
	require.NoError(t, err)

	rel, err := entity.RelationTo[Contractor](meta)
	require.NoError(t, err)
	assert.Equal(t, "UF_CONTRACTOR_ID", rel.FKColumn)
}

func TestInvalidRelationFK(t *testing.T) {
	type Bad struct {
		ID         int64      `bx:"ID;pk;auto;table=t"`
		Contractor Contractor `bx:"rel=Contractor;fk=UF_MISSING"`
	}
	_, err := entity.For[Bad]()
	require.Error(t, err)
}

type ParentContractor struct {
	ID        int64      `bx:"ID;pk;auto"`
	Name      string     `bx:"UF_NAME"`
	Contracts []Contract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID;inverse"`
}

func (ParentContractor) TableName() string { return "contractor" }

func TestInverseRelationMeta(t *testing.T) {
	meta, err := entity.For[ParentContractor]()
	require.NoError(t, err)

	require.Len(t, meta.Relations, 1)
	rel := meta.Relations[0]
	assert.True(t, rel.Inverse)
	assert.Equal(t, "Contracts", rel.Name)
	assert.Equal(t, "UF_CONTRACTOR_ID", rel.FKColumn)

	insertCols := meta.InsertColumns()
	for _, c := range insertCols {
		assert.NotEqual(t, "Contracts", c.Name)
	}

	inverse := meta.InverseRelations()
	require.Len(t, inverse, 1)
	assert.Equal(t, "Contracts", inverse[0].Name)
}

func TestInverseRequiresSlice(t *testing.T) {
	type Bad struct {
		ID        int64    `bx:"ID;pk;auto;table=t"`
		Contracts Contract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID;inverse"`
	}
	_, err := entity.For[Bad]()
	require.Error(t, err)
}

func TestSliceRequiresInverse(t *testing.T) {
	type Bad struct {
		ID        int64      `bx:"ID;pk;auto;table=t"`
		Contracts []Contract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID"`
	}
	_, err := entity.For[Bad]()
	require.Error(t, err)
}

func TestInvalidInverseFK(t *testing.T) {
	type BadParent struct {
		ID        int64      `bx:"ID;pk;auto;table=t"`
		Contracts []Contract `bx:"rel=Contract;fk=UF_MISSING;inverse"`
	}
	_, err := entity.For[BadParent]()
	require.Error(t, err)
}

func TestCircularRelationMeta(t *testing.T) {
	parentMeta, err := entity.For[cycleContractor]()
	require.NoError(t, err)
	childMeta, err := entity.For[cycleContract]()
	require.NoError(t, err)
	require.Len(t, parentMeta.InverseRelations(), 1)
	require.Len(t, childMeta.Relations, 1)
	assert.False(t, childMeta.Relations[0].Inverse)
}

type cycleContractor struct {
	ID        int64           `bx:"ID;pk;auto;table=contractor"`
	Name      string          `bx:"UF_NAME"`
	Contracts []cycleContract `bx:"rel=cycleContract;fk=UF_CONTRACTOR_ID;inverse"`
}

type cycleContract struct {
	ID           int64           `bx:"ID;pk;auto;table=contract"`
	ContractorID int64           `bx:"UF_CONTRACTOR_ID;ref=cycleContractor"`
	Contractor   cycleContractor `bx:"rel=cycleContractor;fk=UF_CONTRACTOR_ID"`
	Name         string          `bx:"UF_NAME"`
}
