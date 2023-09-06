package golang

import (
	"fmt"
	"net"
	"os"
)

type config struct {
	migrations string
	fixtures   string

	withDefaultMigrations bool
	withDefaultFixtures   bool

	// postgres instance information
	instancePort   uint32
	instanceUser   string
	instancePass   string
	instanceDBName string

	hostAddress string
	hostPort    uint32
}

var defaultConfig = &config{
	instancePass:   "postgres",
	instancePort:   15432,
	instanceUser:   "postgres",
	instanceDBName: "postgres",

	hostAddress: "localhost",
	hostPort:    1988,
}

type Option func(*config) error

func WithMigrations(migrations string) Option {
	return func(cfg *config) error {
		cfg.migrations = migrations
		return nil
	}
}

func WithDefaultMigrations() Option {
	return func(cfg *config) error {
		cfg.withDefaultMigrations = true
		return nil
	}
}

func WithFixtures(fixtures string) Option {
	return func(cfg *config) error {
		cfg.fixtures = fixtures
		return nil
	}
}

func WithDefaultFixtures() Option {
	return func(cfg *config) error {
		cfg.withDefaultFixtures = true
		return nil
	}
}

func WithInstance(user, pass, address, dbname string, port uint32) Option {
	return func(cfg *config) error {
		cfg.instanceUser = user
		cfg.instancePass = pass
		cfg.instanceDBName = dbname
		cfg.instancePort = port
		return nil
	}
}

func WithHost(address string, port uint32) Option {
	return func(cfg *config) error {
		cfg.hostAddress = address
		cfg.hostPort = port
		return nil
	}
}

func (c *config) getHostURL() string {
	host := c.hostAddress
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		host = "host.docker.internal"
	}
	return "http://" + net.JoinHostPort(host, fmt.Sprintf("%d", c.hostPort))
}
