package pg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type config struct {
	pass    string
	user    string
	name    string
	port    uint32
	version string

	label string

	withUI bool
	logger io.Writer

	migrationsFiles []string
	fixtureFiles    []string
}

var (
	supportedVersions = map[string]string{
		"10.3.2": "postgis/postgis:10-3.2-alpine",
		"11.2.5": "postgis/postgis:11-2.5-alpine",
		"11.3.2": "postgis/postgis:11-3.2-alpine",
		"12.3.2": "postgis/postgis:12-3.2-alpine",
		"13-3.1": "odidev/postgis:13-3.1-alpine",
		"13.3.2": "postgis/postgis:13-3.2-alpine",
		"14.3.2": "postgis/postgis:14-3.2-alpine",
	}
)

// Option is the type of the functional options for the postgres
type Option func(*config) error

// WithUI applied withUI option to config
func WithUI(withIU bool) Option {
	return func(c *config) error {
		c.withUI = withIU
		return nil
	}
}

// WithLabel applied selected label to config
func WithLabel(label string) Option {
	return func(c *config) error {
		c.label = label
		return nil
	}
}

// WithHost applied selected postgres host to config
func WithHost(user, pass, name string, port uint32) Option {
	return func(c *config) error {
		c.user = user
		c.pass = pass
		c.name = name
		c.port = port
		return nil
	}
}

// WithURI applies the instance connection details found in a database uri to config.
// The database name is deliberately left untouched: administrative statements have to
// run against the maintenance database, not against the one being created or dropped.
func WithURI(uri string) Option {
	return func(c *config) error {
		if uri == "" {
			return nil
		}

		u, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("parse database uri failed: %w", err)
		}

		if u.User != nil {
			if name := u.User.Username(); name != "" {
				c.user = name
			}
			if pass, ok := u.User.Password(); ok {
				c.pass = pass
			}
		}

		if p := u.Port(); p != "" {
			port, err := strconv.ParseUint(p, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid port in database uri %q: %w", uri, err)
			}
			c.port = uint32(port)
		}

		return nil
	}
}

// WithVersion applied selected postgres version to config
func WithVersion(version string) Option {
	vv := strings.TrimSpace(version)
	return func(c *config) error {
		if vv == "" {
			c.version = DefaultVersion
			return nil
		}
		versions := getVersions()
		for _, v := range versions {
			if v == vv {
				c.version = version
				return nil
			}
		}
		return fmt.Errorf("seleced postgres version (%s) is not supported, select one of: %s", vv, strings.Join(versions, ","))
	}
}

func getVersions() []string {
	out := make([]string, 0)
	for k := range supportedVersions {
		out = append(out, k)
	}
	return out
}

// WithLogger applied selected logger to config
func WithLogger(logger io.Writer) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

// WithMigrations applied selected migrations to config
func WithMigrations(path string) Option {
	return func(c *config) error {
		files, err := getFiles(path)
		if err != nil {
			return fmt.Errorf("read migraions failed: %w", err)
		}

		c.migrationsFiles = append(c.migrationsFiles, MigrationFiles(files)...)
		return nil
	}
}

// MigrationFiles drops the down migrations from a list of migration files. Applying
// them alongside the up migrations would undo the schema that was just created.
func MigrationFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f), "down.sql") {
			continue
		}
		out = append(out, f)
	}
	return out
}

// WithFixtures applied selected fixtures to config
func WithFixtures(path string) Option {
	return func(c *config) error {
		files, err := getFiles(path)
		if err != nil {
			return fmt.Errorf("read fixtures failed: %w", err)
		}
		c.fixtureFiles = files
		return nil
	}
}

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
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s not exit", path)
		}
		out = append(out, path)
		return out, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		// only sql files are applied, a directory can hold a readme or a
		// checked in .gitkeep next to the migrations.
		if f.IsDir() || !strings.EqualFold(filepath.Ext(f.Name()), ".sql") {
			continue
		}
		out = append(out, filepath.Join(absPath, f.Name()))
	}

	sort.Strings(out)
	return out, nil
}

// templateName derives a postgres identifier from the content of the given files.
// Hashing the content rather than the paths is what makes the template reusable:
// the api server writes every request's uploads into a fresh temporary directory,
// so paths differ on every call while the migrations themselves do not.
func templateName(files []string) (string, error) {
	h := sha256.New()
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("read migration file (%s) failed: %w", f, err)
		}

		// include the name so that renaming or reordering migrations yields a
		// different template.
		h.Write([]byte(filepath.Base(f)))
		h.Write(b)
	}

	// postgres truncates identifiers at 63 bytes, keep well below that.
	return "dbctl_tpl_" + hex.EncodeToString(h.Sum(nil))[:32], nil
}

func getPostGisImage(version string) string {
	if v, ok := supportedVersions[version]; ok {
		return v
	}
	// fallback to odidev/postgis:13-3.1
	return "odidev/postgis:13-3.1-alpine"
}
