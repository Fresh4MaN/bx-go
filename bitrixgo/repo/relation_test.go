package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"bitrixgo/bitrixgo/entity"
)

type Contractor struct {
	ID   int64  `bx:"ID;pk;auto"`
	Name string `bx:"UF_NAME"`
}

func (Contractor) TableName() string { return "contractor" }

type Contract struct {
	ID           int64      `bx:"ID;pk;auto"`
	ContractorID int64      `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
	Contractor   Contractor `bx:"rel=Contractor;fk=UF_CONTRACTOR_ID"`
	Name         string     `bx:"UF_NAME"`
}

func (Contract) TableName() string { return "contract" }

func TestGetListByFKBuildsFilter(t *testing.T) {
	meta, err := entity.For[Contract]()
	require.NoError(t, err)

	rel, err := meta.ResolveRelation("Contractor")
	require.NoError(t, err)
	require.Equal(t, "UF_CONTRACTOR_ID", rel.FKColumn)
}
