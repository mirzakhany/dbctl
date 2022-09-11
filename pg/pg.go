package pg

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"

	embedpg "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mirzakhany/dbctl/internal/pkg"
	"github.com/spf13/cobra"
)

type Postgres struct {
	cfg config
}

func New(options ...Option) (*Postgres, error) {
	// create postgres with default values
	pg := &Postgres{cfg: config{
		pass:    "postgres",
		user:    "postgres",
		name:    "postgres",
		port:    5432,
		version: "14.3.0",
	}}

	for _, o := range options {
		if err := o(&pg.cfg); err != nil {
			return nil, err
		}
	}
	return pg, nil
}

func (p *Postgres) Start() error {
	ctx := pkg.ContextWithOsSignal()

	config := embedpg.DefaultConfig().
		Locale("en_US.UTF-8").
		Username(p.cfg.user).
		Password(p.cfg.pass).
		Database(p.cfg.name).
		Version(embedpg.PostgresVersion(p.cfg.version)).
		Port(p.cfg.port).
		Logger(p.cfg.logger)

	database := embedpg.NewDatabase(config)
	log.Printf("Starting postgres version %s on port %d ...\n", p.cfg.version, p.cfg.port)
	if err := database.Start(); err != nil {
		return err
	}

	log.Println("Postgres is up an running")

	// run migrations if exist
	if len(strings.TrimSpace(p.cfg.migrationsPath)) != 0 {
		log.Println("Applying migration files")
		if err := p.runMigrations(); err != nil {
			return err
		}
	}

	// print connection url
	log.Printf("Database uri is: %q\n", p.URI())

	defer func() {
		log.Println("Shutdown signal received, stopping database")
		if err := database.Stop(); err != nil {
			cobra.CheckErr(err)
		}
	}()

	<-ctx.Done()
	return nil
}

func (p *Postgres) URI() string {
	host := net.JoinHostPort("localhost", strconv.Itoa(int(p.cfg.port)))
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(p.cfg.user, p.cfg.pass), Host: host, Path: p.cfg.name, RawQuery: "sslmode=disable"}).String()
}

func (p *Postgres) printURI() {
	log.Printf("postgres database is running on: %q", p.URI())
}

func (p *Postgres) runMigrations() error {
	m, err := migrate.New(p.cfg.migrationsPath, p.URI())
	if err != nil {
		return fmt.Errorf("run migrations failed %w", err)
	}
	return m.Steps(1)
}
