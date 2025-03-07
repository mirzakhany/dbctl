package mongodb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type config struct {
	user    string
	pass    string
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
		"6.0": "mongo:6.0",
		"7.0": "mongo:7.0", // Latest stable version
	}
)

// Option is the type of the functional options for MongoDB
type Option func(*config) error

// WithUI applies withUI option to config
func WithUI(withUI bool) Option {
	return func(c *config) error {
		c.withUI = withUI
		return nil
	}
}

// WithLabel applies selected label to config
func WithLabel(label string) Option {
	return func(c *config) error {
		c.label = label
		return nil
	}
}

// WithHost applies selected MongoDB host to config
func WithHost(user, pass, name string, port uint32) Option {
	return func(c *config) error {
		c.user = user
		c.pass = pass
		c.name = name
		c.port = port
		return nil
	}
}

// WithVersion applies selected MongoDB version to config
func WithVersion(version string) Option {
	vv := strings.TrimSpace(version)
	return func(c *config) error {
		if vv == "" {
			c.version = "7.0" // Default to latest version
			return nil
		}
		versions := getVersions()
		for _, v := range versions {
			if v == vv {
				c.version = version
				return nil
			}
		}
		return fmt.Errorf("selected MongoDB version (%s) is not supported, select one of: %s", vv, strings.Join(versions, ","))
	}
}

func getVersions() []string {
	out := make([]string, 0)
	for k := range supportedVersions {
		out = append(out, k)
	}
	return out
}

// WithLogger applies selected logger to config
func WithLogger(logger io.Writer) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

// WithMigrations applies selected migrations to config
func WithMigrations(path string) Option {
	return func(c *config) error {
		files, err := getFiles(path)
		if err != nil {
			return fmt.Errorf("read migrations failed: %w", err)
		}

		c.migrationsFiles = files
		return nil
	}
}

// WithFixtures applies selected fixtures to config
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
		out = append(out, filepath.Join(absPath, f.Name()))
	}

	sort.Strings(out)
	return out, nil
}

func getMongoDBImage(version string) string {
	if v, ok := supportedVersions[version]; ok {
		return v
	}
	// fallback to mongo:7.0
	return "mongo:7.0"
}
