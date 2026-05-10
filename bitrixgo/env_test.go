package bitrixgo_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitrixgo/bitrixgo"
)

func TestDSNFromEnvFull(t *testing.T) {
	t.Setenv("BITRIX_DSN", "user:pass@tcp(db.example.com:3307)/bitrix?parseTime=true")
	t.Setenv("BITRIX_DB_HOST", "")

	dsn, err := bitrixgo.DSNFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "user:pass@tcp(db.example.com:3307)/bitrix?parseTime=true", dsn)
}

func TestDSNFromEnvParts(t *testing.T) {
	t.Setenv("BITRIX_DSN", "")
	t.Setenv("BITRIX_DB_HOST", "10.0.0.5")
	t.Setenv("BITRIX_DB_PORT", "3306")
	t.Setenv("BITRIX_DB_USER", "bitrix")
	t.Setenv("BITRIX_DB_PASSWORD", "secret")
	t.Setenv("BITRIX_DB_NAME", "sitemanager")

	dsn, err := bitrixgo.DSNFromEnv()
	require.NoError(t, err)
	assert.Contains(t, dsn, "bitrix:secret@tcp(10.0.0.5:3306)/sitemanager")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "charset=utf8mb4")
}

func TestDSNFromEnvMissing(t *testing.T) {
	for _, key := range []string{
		"BITRIX_DSN", "BITRIX_DB_HOST", "BITRIX_DB_USER", "BITRIX_DB_PASSWORD", "BITRIX_DB_NAME",
	} {
		t.Setenv(key, "")
	}

	_, err := bitrixgo.DSNFromEnv()
	require.Error(t, err)
}

func TestLoadEnvMissingFile(t *testing.T) {
	err := bitrixgo.LoadEnv(".env.does-not-exist")
	require.NoError(t, err)
}

func TestLoadEnvFromFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "env-")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("BITRIX_DB_HOST=remote.host\nBITRIX_DB_USER=app\n")
	require.NoError(t, err)

	require.NoError(t, bitrixgo.LoadEnv(f.Name()))
	assert.Equal(t, "remote.host", os.Getenv("BITRIX_DB_HOST"))
	assert.Equal(t, "app", os.Getenv("BITRIX_DB_USER"))
}

func TestTablePrefixFromEnv(t *testing.T) {
	t.Setenv("BITRIX_TABLE_PREFIX", "b_")
	assert.Equal(t, "b_", bitrixgo.TablePrefixFromEnv())
}
