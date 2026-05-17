package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitrixgo/bitrixgo/entity"
)

type Consignee struct {
	ID      int64  `bx:"ID;pk;auto"`
	Address string `bx:"UF_ADDRESS"`
	Kpp     string `bx:"UF_KPP"`
	Inn     string `bx:"UF_INN"`
	Name    string `bx:"UF_NAME"`
	Guid    string `bx:"UF_GUID"`
}

func (Consignee) TableName() string { return "consignee" }

func TestFor(t *testing.T) {
	meta, err := entity.For[Consignee]()
	require.NoError(t, err)

	assert.Equal(t, "consignee", meta.Table)
	assert.Equal(t, "ID", meta.PrimaryKey)
	assert.Len(t, meta.Fields, 6)

	cols := meta.InsertColumns()
	assert.Len(t, cols, 5)
	for _, c := range cols {
		assert.NotEqual(t, "ID", c.Column)
	}
}

func TestForTableTag(t *testing.T) {
	type Tagged struct {
		ID   int64  `bx:"ID;pk;auto;table=custom_table"`
		Name string `bx:"NAME"`
	}
	meta, err := entity.For[Tagged]()
	require.NoError(t, err)
	assert.Equal(t, "custom_table", meta.Table)
}

func TestForMissingTable(t *testing.T) {
	type Bad struct {
		ID int64 `bx:"ID;pk"`
	}
	_, err := entity.For[Bad]()
	require.Error(t, err)
}

func TestBoolIntConversion(t *testing.T) {
	type Item struct {
		ID     int64 `bx:"ID;pk;auto;table=t"`
		Active bool  `bx:"UF_ACTIVE"`
	}
	meta, err := entity.For[Item]()
	require.NoError(t, err)

	item := Item{Active: true}
	cols := meta.InsertColumns()
	vals, err := entity.ValuesFromEntity(meta, &item, cols)
	require.NoError(t, err)
	assert.Equal(t, int64(1), vals[0])

	item.Active = false
	vals, err = entity.ValuesFromEntity(meta, &item, cols)
	require.NoError(t, err)
	assert.Equal(t, int64(0), vals[0])
}

func TestBoolYNConversion(t *testing.T) {
	type Item struct {
		ID     int64 `bx:"ID;pk;auto;table=t"`
		Active bool  `bx:"UF_ACTIVE;boolyn"`
	}
	meta, err := entity.For[Item]()
	require.NoError(t, err)

	item := Item{Active: true}
	cols := meta.InsertColumns()
	vals, err := entity.ValuesFromEntity(meta, &item, cols)
	require.NoError(t, err)
	assert.Equal(t, "Y", vals[0])
}
