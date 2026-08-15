package dbctlgo

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

// config is the client configuration.
type config struct {
	migrations          string
	migrationsFileRegex string

	fixtures string

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

// Environment variables read when no option overrides them, so that a project can
// point its tests at its own dbctl server without touching the test code.
const (
	// EnvHost is the address of the dbctl api server
	EnvHost = "DBCTL_HOST"
	// EnvPort is the port of the dbctl api server
	EnvPort = "DBCTL_PORT"
)

// The instance details are left empty on purpose: the server knows the defaults of
// each database type and fills in the missing ones. Sending the postgres defaults
// for every type would make dbctl try to log into redis as "postgres".
var defaultConfig = &config{
	hostAddress: "localhost",
	hostPort:    1988,
}

// fromEnv applies the environment to a config, before any explicit option.
func (c *config) fromEnv() error {
	if host := os.Getenv(EnvHost); host != "" {
		c.hostAddress = host
	}

	if port := os.Getenv(EnvPort); port != "" {
		p, err := strconv.ParseUint(port, 10, 32)
		if err != nil {
			return fmt.Errorf("%s is not a valid port number: %q", EnvPort, port)
		}
		c.hostPort = uint32(p)
	}

	return nil
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
// The address is accepted for backwards compatibility and ignored: the dbctl
// server always reaches the instances running next to it.
func WithInstance(user, pass, address, dbname string, port uint32) Option { //nolint:revive
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

// WithMigrationsFileRegex configures the client to use the given regex to match what migrations file to apply.
func WithMigrationsFileRegex(regex string) Option {
	return func(cfg *config) error {
		cfg.migrationsFileRegex = regex
		return nil
	}
}

// getHostURL returns the host url.
func (c *config) getHostURL() string {
	return "http://" + net.JoinHostPort(c.hostAddress, fmt.Sprintf("%d", c.hostPort))
}
