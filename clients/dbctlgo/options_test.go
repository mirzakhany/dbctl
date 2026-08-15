package dbctlgo

import "testing"

// Options used to be applied to the package level default, which meant the
// migrations of one test silently became the migrations of every test after it.
func TestOptionsDoNotLeakIntoTheDefaults(t *testing.T) {
	before := *defaultConfig

	// fails while reading the migrations directory, that is enough to have the
	// options applied without needing a running dbctl server.
	if _, err := CreateDB(DatabasePostgres,
		WithMigrations("./there-is-no-such-directory"),
		WithFixtures("./neither-is-there-this-one"),
		WithMigrationsFileRegex("^.*up.sql$"),
		WithHost("127.0.0.1", 1),
	); err == nil {
		t.Fatal("expected creating a database with a missing migrations directory to fail")
	}

	if *defaultConfig != before {
		t.Fatalf("options leaked into the package defaults: %+v", *defaultConfig)
	}
}

func TestInvalidDatabaseType(t *testing.T) {
	if _, err := CreateDB("mysql"); err != ErrInvalidDatabaseType {
		t.Fatalf("expected ErrInvalidDatabaseType, got %v", err)
	}
}
