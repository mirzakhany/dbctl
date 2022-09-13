package pg

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	embedpg "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v4"

	// golang migration postgres driver
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// golang migration file driver
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/pkg"
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
	var closeFunc func(ctx context.Context) error

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
	if err := p.runMigrations(); err != nil {
		return err
	}

	// run apply fixtures if exist
	if err := p.applyFixtures(ctx); err != nil {
		return err
	}

	// print connection url
	log.Printf("Database uri is: %q\n", p.URI())

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	return closeFunc(shutdownCtx)
}

func (p *Postgres) startUsingNative() (func(ctx context.Context) error, error) {
	config := embedpg.DefaultConfig().
		Locale("en_US.UTF-8").
		Username(p.cfg.user).
		Password(p.cfg.pass).
		Database(p.cfg.name).
		Version(embedpg.PostgresVersion(p.cfg.version)).
		Port(p.cfg.port).
		Logger(p.cfg.logger)

	database := embedpg.NewDatabase(config)
	closeFunc := func(ctx context.Context) error {
		return database.Stop()
	}

	if err := database.Start(); err != nil {
		return closeFunc, err
	}

	return closeFunc, nil
}

func (p *Postgres) startUsingDocker(ctx context.Context) (func(ctx context.Context) error, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	port := strconv.Itoa(int(p.cfg.port))
	pg, err := container.Run(ctx, container.Request{
		Image: getPostGisImage(p.cfg.version),
		Env: map[string]string{
			"POSTGRES_PASSWORD": p.cfg.pass,
			"POSTGRES_USER":     p.cfg.user,
			"POSTGRES_DB":       p.cfg.name,
		},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		ExposedPorts: []string{fmt.Sprintf("%s:5432/tcp", port)},
		Name:         fmt.Sprintf("dbctl_%d_%d", time.Now().Unix(), rnd.Uint64()),
	})
	if err != nil {
		return nil, err
	}

	closeFunc := func(ctx context.Context) error {
		return pg.Terminate(ctx)
	}

	return closeFunc, waitForPostgres(ctx, p.URI())
}

func (p *Postgres) URI() string {
	host := net.JoinHostPort("localhost", strconv.Itoa(int(p.cfg.port)))
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(p.cfg.user, p.cfg.pass), Host: host, Path: p.cfg.name, RawQuery: "sslmode=disable"}).String()
}

func (p *Postgres) runMigrations() error {
	if len(p.cfg.migrationsPath) == 0 {
		return nil
	}

	log.Println("Applying migrations ...")
	m, err := migrate.New(p.cfg.migrationsPath, p.URI())
	if err != nil {
		return fmt.Errorf("run migrations failed %w", err)
	}
	return m.Up()
}

func (p *Postgres) applyFixtures(ctx context.Context) error {
	if len(p.cfg.fixtureFiles) == 0 {
		return nil
	}

	log.Println("Applying fixtures ...")
	conn, err := pgx.Connect(ctx, p.URI())
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	for _, f := range p.cfg.fixtureFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read fixture file (%s) failed: %w", f, err)
		}

		if _, err := conn.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("applying fixture file (%s) failed: %w", f, err)
		}
	}
	return nil
}

func waitForPostgres(ctx context.Context, url string) error {
	log.Println("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	for range ticker.C {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			if err == context.DeadlineExceeded {
				return err
			}
		} else {
			_ = conn.Close(context.Background())
			return nil
		}
	}
	return nil
}
