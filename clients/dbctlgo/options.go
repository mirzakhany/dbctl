package dbctlgo

import (
	"fmt"
	"net"
)

// config is the client configuration.
type config struct {
	migrations string
	fixtures   string

	// whether or not to use default migrations/fixtures loaded when dbctl started
	withDefaultMigrations bool

	// postgres instance information
	instancePort   uint32
	instanceUser   string
	instancePass   string
	instanceDBName string

	// host and port of the host, where the dbctl testing server is running
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

// Option is a function that configures the client.
type Option func(*config) error

// WithMigrations configures the client to use the given migrations.
func WithMigrations(migrations string) Option {
	return func(cfg *config) error {
		cfg.migrations = migrations
		return nil
	}
}

// WithDefaultMigrations configures the client to use the default migrations.
func WithDefaultMigrations() Option {
	return func(cfg *config) error {
		cfg.withDefaultMigrations = true
		return nil
	}
}

// WithFixtures configures the client to use the given fixtures.
func WithFixtures(fixtures string) Option {
	return func(cfg *config) error {
		cfg.fixtures = fixtures
		return nil
	}
}

// WithInstance configures the client to use the given postgres instance.
func WithInstance(user, pass, address, dbname string, port uint32) Option {
	return func(cfg *config) error {
		cfg.instanceUser = user
		cfg.instancePass = pass
		cfg.instanceDBName = dbname
		cfg.instancePort = port
		return nil
	}
}

// WithHost configures the client to use the given host.
func WithHost(address string, port uint32) Option {
	return func(cfg *config) error {
		cfg.hostAddress = address
		cfg.hostPort = port
		return nil
	}
}

// getHostURL returns the host url.
func (c *config) getHostURL() string {
	return "http://" + net.JoinHostPort(c.hostAddress, fmt.Sprintf("%d", c.hostPort))
}
