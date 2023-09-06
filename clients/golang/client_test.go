package golang

import (
	"database/sql"

	"testing"
	// golang postgres driver
	_ "github.com/lib/pq"
)

func TestMustCreateDB(t *testing.T) {
	uri := MustCreatePostgresDB(t, WithDefaultMigrations())
	if uri == "" {
		t.Fatal("url is empty")
	}

	conn, err := sql.Open("postgres", uri)
	if err != nil {
		t.Fatal(err)
	}

	if err := conn.Ping(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// do something with conn
	res, err := conn.Exec("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}

	if re, _ := res.RowsAffected(); re != 1 {
		t.Fatal("expected 1 rows affected")
	}
}
