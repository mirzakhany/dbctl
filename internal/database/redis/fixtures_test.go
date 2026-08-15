package redis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`SET key value`, []string{"SET", "key", "value"}},
		{`SET greeting "hello world"`, []string{"SET", "greeting", "hello world"}},
		{`SET greeting 'hello world'`, []string{"SET", "greeting", "hello world"}},
		{`  RPUSH   list   a   b  `, []string{"RPUSH", "list", "a", "b"}},
		{`SET empty ""`, []string{"SET", "empty", ""}},
		{`HSET user:1 name "Ada Lovelace" born 1815`, []string{"HSET", "user:1", "name", "Ada Lovelace", "born", "1815"}},
	}

	for _, tc := range cases {
		got, err := splitCommand(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}

		if len(got) != len(tc.want) {
			t.Fatalf("%q: expected %q, got %q", tc.in, tc.want, got)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: expected %q, got %q", tc.in, tc.want, got)
			}
		}
	}

	if _, err := splitCommand(`SET key "unbalanced`); err == nil {
		t.Fatal("expected an unbalanced quote to be rejected")
	}
}

func TestGetFilesPicksFixtureFiles(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"002_second.redis":      "SET b 2",
		"001_first.redis":       "SET a 1",
		"nested/003_third.lua":  `redis.call("SET", "c", 3)`,
		"nested/004_fourth.txt": "SET d 4",
		"README.md":             "not a fixture",
	} {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := getFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"001_first.redis", "002_second.redis", "nested/003_third.lua", "nested/004_fourth.txt"}
	if len(files) != len(want) {
		t.Fatalf("expected %v, got %v", want, files)
	}

	for i, f := range files {
		rel, err := filepath.Rel(dir, f)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.ToSlash(rel) != want[i] {
			t.Fatalf("expected %v, got %s at %d", want, rel, i)
		}
	}
}
