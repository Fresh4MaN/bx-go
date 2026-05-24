package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Fresh4MaN/bx-go/bitrixgo/entity"
)

func TestScanBoolFromInt(t *testing.T) {
	type Item struct {
		ID     int64 `bx:"ID;pk;auto;table=t"`
		Active bool  `bx:"UF_ACTIVE"`
	}
	meta, err := entity.For[Item]()
	require.NoError(t, err)

	cols, vals, err := entity.ValuesFromMap(meta, map[string]any{"UF_ACTIVE": int64(1)})
	require.NoError(t, err)
	assert.Equal(t, "UF_ACTIVE", cols[0])
	assert.Equal(t, int64(1), vals[0])

	cols, vals, err = entity.ValuesFromMap(meta, map[string]any{"UF_ACTIVE": false})
	require.NoError(t, err)
	assert.Equal(t, int64(0), vals[0])
}
