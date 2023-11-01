package redis

import (
	"fmt"
	"io"
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
