package pg

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go/wait"

	embedpg "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"

	// golang migration postgres driver
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// golang migration file driver
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/pkg"
	"github.com/testcontainers/testcontainers-go"
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
	log.Printf("Starting postgres version %s on port %d ...\n", p.cfg.version, p.cfg.port)

	var err error
	var closeFunc func() error

	if p.cfg.useDockerEngine {
		closeFunc, err = p.startUsingDocker(ctx)
	} else {
		closeFunc, err = p.startUsingNative()
	}
	if err != nil {
		return err
	}

	log.Println("Postgres is up and running")
	// run migrations if exist
	if len(strings.TrimSpace(p.cfg.migrationsPath)) != 0 {
		log.Println("Applying migration files")
		if err := p.runMigrations(); err != nil {
			return err
		}
	}

	// print connection url
	log.Printf("Database uri is: %q\n", p.URI())

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping database")
	return closeFunc()
}

func (p *Postgres) startUsingNative() (func() error, error) {
	config := embedpg.DefaultConfig().
		Locale("en_US.UTF-8").
		Username(p.cfg.user).
		Password(p.cfg.pass).
		Database(p.cfg.name).
		Version(embedpg.PostgresVersion(p.cfg.version)).
		Port(p.cfg.port).
		Logger(p.cfg.logger)

	database := embedpg.NewDatabase(config)
	if err := database.Start(); err != nil {
		return func() error {
			return database.Stop()
		}, err
	}

	return func() error {
		return database.Stop()
	}, nil
}

func (p *Postgres) startUsingDocker(ctx context.Context) (func() error, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}
	port := strconv.Itoa(int(p.cfg.port))
	pg, err := container.Run(ctx, testcontainers.ContainerRequest{
		Image: getPostGisImage(p.cfg.version),
		Env: map[string]string{
			"POSTGRES_PASSWORD": p.cfg.pass,
			"POSTGRES_USER":     p.cfg.user,
			"POSTGRES_DB":       p.cfg.name,
		},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		ExposedPorts: []string{fmt.Sprintf("%s:5432/tcp", port)},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
		Name:         fmt.Sprintf("dbctl_%d_%d", time.Now().Unix(), rnd.Uint64()),
	})
	if err != nil {
		return nil, err
	}

	closeFunc := func() error { return pg.Terminate(ctx) }
	return closeFunc, nil
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
