package redis

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type config struct {
	pass    string
	user    string
	port    uint32
	dbIndex int
	version string

	label string

	detached bool

	logger io.Writer

	fixtureFiles []string
}

// WithFixtures applies the selected fixtures to config
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

var (
	supportedDockerVersions = map[string]string{
		"7.0.4": "redis:7.0.4-bullseye",
	}
)

type Option func(*config) error

func WithHost(user, pass string, dbIndex int, port uint32) Option {
	return func(c *config) error {
		c.user = user
		c.pass = pass
		c.port = port
		c.dbIndex = dbIndex
		return nil
	}
}

// WithURI applies the instance connection details found in a database uri to config.
// The database index is deliberately left untouched: administrative commands run
// against index 0, where dbctl keeps its bookkeeping keys.
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

func WithLabel(label string) Option {
	return func(c *config) error {
		c.label = label
		return nil
	}
}

// WithVersion applied selected postgres version to config
func WithVersion(version string) Option {
	vv := strings.TrimSpace(version)
	return func(c *config) error {
		if vv == "" {
			c.version = "7.0.4"
			return nil
		}
		versions := getVersions()
		for _, v := range versions {
			if v == vv {
				c.version = version
				return nil
			}
		}
		return fmt.Errorf("seleced redis version (%s) is not supported, select one of: %s", vv, strings.Join(versions, ","))
	}
}

func getVersions() []string {
	out := make([]string, 0)
	for k := range supportedDockerVersions {
		out = append(out, k)
	}
	return out
}

func WithLogger(logger io.Writer) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

func getRedisImage(version string) string {
	if v, ok := supportedDockerVersions[version]; ok {
		return v
	}
	// fallback to redis:7.0.4-bullseye
	return "redis:7.0.4-bullseye"
}
