package golang

import (
	"database/sql"

	"testing"
	// golang postgres driver
	_ "github.com/lib/pq"
)

func TestMustCreateDB(t *testing.T) {
	uri := MustCreatePostgresDB(t, WithMigrations("./test_data"))
	if uri == "" {
		t.Fatal("url is empty")
	}

	t.Log("uri:", uri)

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
	res, err := conn.Exec("insert into foo (name) values ('bar')")
	if err != nil {
		t.Fatal(err)
	}

	if re, _ := res.RowsAffected(); re != 1 {
		t.Fatal("expected 1 rows affected")
	}

	var name string
	if err := conn.QueryRow("select name from foo").Scan(&name); err != nil {
		t.Fatal(err)
	}

	if name != "bar" {
		t.Fatalf("expected name to be bar, got %s", name)
	}
}
