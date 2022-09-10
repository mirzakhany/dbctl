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

	migrationsPath string
}

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
	validVersions := []string{"14.3.0", "13.7.0", "12.11.0", "11.16.0", "10.21.0", "9.6.24"}
	vv := strings.TrimSpace(version)
	return func(c *config) error {
		if vv == "" {
			return nil
		}
		for _, v := range validVersions {
			if v == vv {
				c.version = version
				return nil
			}
		}
		return fmt.Errorf("seleced postgres version (%s) is not supported, select one of: %s", vv, strings.Join(validVersions, ","))
	}
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
