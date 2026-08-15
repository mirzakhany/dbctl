package redis

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gomodule/redigo/redis"
	"github.com/mirzakhany/dbctl/internal/logger"
	"github.com/mirzakhany/dbctl/internal/utils"
)

// applyFixtures loads the given files into the database the connection is on.
//
// Two formats are supported, picked by extension:
//
//	.lua           evaluated as a script, the way redis-cli --eval does
//	anything else  one redis command per line, in redis-cli syntax
func applyFixtures(conn redis.Conn, files []string) error {
	if len(files) == 0 {
		return nil
	}

	logger.Info(fmt.Sprintf("Applying %d fixture files ...", len(files)))

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read fixture file (%s) failed: %w", f, err)
		}

		if strings.EqualFold(filepath.Ext(f), ".lua") {
			if _, err := redis.NewScript(0, string(content)).Do(conn); err != nil && !isNil(err) {
				return fmt.Errorf("applying fixture file (%s) failed: %w", f, err)
			}
			continue
		}

		if err := applyCommands(conn, string(content)); err != nil {
			return fmt.Errorf("applying fixture file (%s) failed: %w", f, err)
		}
	}

	return nil
}

// applyCommands runs one redis command per line. Blank lines and lines starting
// with # are ignored so fixtures can be commented.
func applyCommands(conn redis.Conn, content string) error {
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		args, err := splitCommand(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
		if len(args) == 0 {
			continue
		}

		params := make([]interface{}, 0, len(args)-1)
		for _, a := range args[1:] {
			params = append(params, a)
		}

		if _, err := conn.Do(args[0], params...); err != nil && !isNil(err) {
			return fmt.Errorf("line %d (%s): %w", i+1, args[0], err)
		}
	}

	return nil
}

// splitCommand splits a command line into its arguments, keeping quoted values
// together so that values with spaces survive.
func splitCommand(line string) ([]string, error) {
	var (
		args    []string
		current strings.Builder
		quote   rune
		started bool
	)

	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
			started = true
		case unicode.IsSpace(r):
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(r)
			started = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unbalanced %q in %q", string(quote), line)
	}

	if started {
		args = append(args, current.String())
	}

	return args, nil
}

// isNil reports whether an error is redigo's "no reply", which scripts returning
// nothing produce.
func isNil(err error) bool {
	return err == redis.ErrNil
}

// getFiles returns the fixture files in path and its subdirectories, sorted by
// path so their order is predictable.
func getFiles(path string) ([]string, error) {
	if len(path) == 0 {
		return nil, nil
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("get path information failed, %w", err)
	}

	out := make([]string, 0)

	if !stat.IsDir() {
		return append(out, path), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(absPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// a fixtures directory can hold a readme next to the fixtures
		if d.IsDir() || !utils.OneOf(strings.ToLower(filepath.Ext(d.Name())), ".redis", ".txt", ".lua") {
			return nil
		}

		out = append(out, p)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(out)
	return out, nil
}
