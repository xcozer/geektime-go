package sql_demo

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDB(t *testing.T)  {
	db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
	require.NoError(t, err)
	db.Ping()
	// 这里你就可以用 DB 了
	// sql.OpenDB()
}