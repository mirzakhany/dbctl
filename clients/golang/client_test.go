package golang

import (
	"database/sql"

	"testing"
	// golang postgres driver
	_ "github.com/lib/pq"
)

func TestMustCreateDB(t *testing.T) {
	uri := MustCreatePostgresDB(t, WithMigrations("./test_sql/schema"))
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
	res, err := conn.Exec("insert into foo (name) values ('test-must-create-db')")
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

	if name != "test-must-create-db" {
		t.Fatalf("expected name to be test-must-create-db, got %s", name)
	}
}

func TestWithFixtures(t *testing.T) {
	uri := MustCreatePostgresDB(t, WithMigrations("./test_sql/schema"), WithFixtures("./test_sql/fixtures"))
	if uri == "" {
		t.Fatal("url is empty")
	}

	t.Log("uri:", uri)

	conn, err := sql.Open("postgres", uri)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// select all from foo
	rows, err := conn.Query("select name from foo")
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}

		names = append(names, name)
	}

	if len(names) != 3 {
		t.Fatal("expected 3 rows")
	}

	expected := []string{"foo", "bar", "baz"}
	for _, name := range names {
		if !contains(expected, name) {
			t.Fatalf("expected name to be one of %v, got %s", expected, name)
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
