package pg

import (
	"fmt"
	"io"
	"strings"
)

type config struct {
	pass    string
	user    string
	name    string
	port    uint32
	version string

	logger io.Writer

	migrationsPath  string
	useDockerEngine bool
}

var (
	supportedNativeVersions = []string{"14.3.0", "13.7.0", "12.11.0", "11.16.0", "10.21.0", "9.6.24"}
	supportedDockerVersions = map[string]string{
		"10.3.2": "postgis/postgis:10-3.2-alpine",
		"11.2.5": "postgis/postgis:11-2.5-alpine",
		"11.3.2": "postgis/postgis:11-3.2-alpine",
		"12.3.2": "postgis/postgis:12-3.2-alpine",
		"13.3.2": "postgis/postgis:13-3.2-alpine",
		"14.3.2": "postgis/postgis:14-3.2-alpine",
	}
)

type Option func(*config) error

func WithHost(user, pass, name string, port uint32) Option {
	return func(c *config) error {
		c.user = user
		c.pass = pass
		c.name = name
		c.port = port
		return nil
	}
}

// WithVersion applied selected postgres version to config
func WithVersion(version string) Option {
	vv := strings.TrimSpace(version)
	return func(c *config) error {
		if vv == "" {
			if c.useDockerEngine {
				c.version = "14.3.2"
			} else {
				c.version = "14.3.0"
			}
			return nil
		}
		versions := getVersions(c.useDockerEngine)
		for _, v := range versions {
			if v == vv {
				c.version = version
				return nil
			}
		}
		return fmt.Errorf("seleced postgres version (%s) is not supported, select one of: %s", vv, strings.Join(versions, ","))
	}
}

func getVersions(useDocker bool) []string {
	out := make([]string, 0)
	if useDocker {
		for k := range supportedDockerVersions {
			out = append(out, k)
		}
		return out
	}
	return supportedNativeVersions
}

func WithLogger(logger io.Writer) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

func WithMigrations(path string) Option {
	return func(c *config) error {
		v := strings.TrimSpace(path)
		if len(v) == 0 {
			return nil
		}

		if v[0] == '.' {
			c.migrationsPath = "file://" + path[2:]
		} else {
			c.migrationsPath = "file:///" + path[1:]
		}

		return nil
	}
}

func WithDockerEngine(useDocker bool) Option {
	return func(c *config) error {
		c.useDockerEngine = useDocker
		return nil
	}
}

func getPostGisImage(version string) string {
	if v, ok := supportedDockerVersions[version]; ok {
		return v
	}
	// fallback to the latest version
	return "postgis/postgis:14-3.2-alpine"
}
