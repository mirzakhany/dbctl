package pg

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestTemplateNameFollowsContentNotPaths(t *testing.T) {
	files := map[string]string{"001_init.up.sql": "create table foo(id int);"}

	// the api server writes every request's uploads into a fresh directory, the
	// template has to stay the same anyway or migrations run on every request.
	first, err := getFiles(writeFiles(t, files))
	if err != nil {
		t.Fatal(err)
	}

	second, err := getFiles(writeFiles(t, files))
	if err != nil {
		t.Fatal(err)
	}

	firstName, err := templateName(first)
	if err != nil {
		t.Fatal(err)
	}

	secondName, err := templateName(second)
	if err != nil {
		t.Fatal(err)
	}

	if firstName != secondName {
		t.Fatalf("same migrations produced different templates: %s != %s", firstName, secondName)
	}

	if len(firstName) > 63 {
		t.Fatalf("template name is longer than a postgres identifier: %d", len(firstName))
	}

	changed, err := getFiles(writeFiles(t, map[string]string{"001_init.up.sql": "create table bar(id int);"}))
	if err != nil {
		t.Fatal(err)
	}

	changedName, err := templateName(changed)
	if err != nil {
		t.Fatal(err)
	}

	if changedName == firstName {
		t.Fatal("different migrations produced the same template name")
	}
}

func TestGetFilesOnlyReturnsSQLFiles(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"001_init.up.sql": "select 1;",
		"README.md":       "not sql",
		".gitkeep":        "",
	})

	files, err := getFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || filepath.Base(files[0]) != "001_init.up.sql" {
		t.Fatalf("expected only the sql file, got %v", files)
	}
}

func TestMigrationFilesDropsDownMigrations(t *testing.T) {
	in := []string{"/m/001_init.up.sql", "/m/002_users.DOWN.sql", "/m/002_users.up.sql"}

	got := MigrationFiles(in)
	if len(got) != 2 {
		t.Fatalf("expected the down migration to be dropped, got %v", got)
	}

	for _, f := range got {
		if filepath.Base(f) == "002_users.DOWN.sql" {
			t.Fatalf("down migration was kept: %v", got)
		}
	}
}

func TestQuoteIdentifier(t *testing.T) {
	if got := quoteIdentifier(`db"x`); got != `"db""x"` {
		t.Fatalf("unexpected quoting: %s", got)
	}
}

func TestWithURIKeepsMaintenanceDatabase(t *testing.T) {
	p, err := New(WithURI("postgres://someone:secret@localhost:5455/dbctl_123?sslmode=disable"))
	if err != nil {
		t.Fatal(err)
	}

	if p.cfg.user != "someone" || p.cfg.pass != "secret" || p.cfg.port != 5455 {
		t.Fatalf("instance details were not taken from the uri: %+v", p.cfg)
	}

	// dropping a database requires a connection to another one
	if p.cfg.name != DefaultName {
		t.Fatalf("expected the maintenance database, got %s", p.cfg.name)
	}
}
