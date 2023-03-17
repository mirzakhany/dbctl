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

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v4"

	// golang migration postgres driver
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// golang migration file driver
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
)

var _ database.Database = (*Postgres)(nil)

type Postgres struct {
	containerID string
	cfg         config
}

func New(options ...Option) (*Postgres, error) {
	// create postgres with default values
	pg := &Postgres{cfg: config{
		pass:    "postgres",
		user:    "postgres",
		name:    "postgres",
		port:    15432,
		version: "14.3.0",
	}}

	for _, o := range options {
		if err := o(&pg.cfg); err != nil {
			return nil, err
		}
	}
	return pg, nil
}

func (p *Postgres) Start(ctx context.Context, detach bool) error {
	log.Printf("Starting postgres version %s on port %d ...\n", p.cfg.version, p.cfg.port)

	closeFunc, err := p.startUsingDocker(ctx, 20*time.Second)
	if err != nil {
		return err
	}

	log.Println("Postgres is up and running")
	// run migrations if exist
	if err := RunMigrations(p.cfg.migrationsPath, p.URI()); err != nil {
		return err
	}

	// run apply fixtures if exist
	if err := ApplyFixtures(ctx, p.cfg.fixtureFiles, p.URI()); err != nil {
		return err
	}

	// print connection url
	log.Printf("Database uri is: %q\n", p.URI())

	// detach and stop cli if asked
	p.cfg.detached = detach
	if detach {
		return nil
	}

	<-ctx.Done()
	log.Println("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	return closeFunc(shutdownCtx)
}

func (p *Postgres) Stop(ctx context.Context) error {
	return container.TerminateByID(ctx, p.containerID)
}

func (p *Postgres) WaitForStart(ctx context.Context, timeout time.Duration) error {
	log.Println("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for range ticker.C {
		conn, err := pgx.Connect(ctx, p.URI())
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

func Instances(ctx context.Context) ([]database.Info, error) {
	l, err := container.List(ctx, map[string]string{database.LabelType: database.LabelPostgres})
	if err != nil {
		return nil, err
	}

	out := make([]database.Info, 0, len(l))
	for _, c := range l {
		out = append(out, database.Info{
			ID:     c.ID,
			Type:   c.Name,
			Status: database.Running,
		})
	}
	return out, nil
}

func (p *Postgres) startUsingDocker(ctx context.Context, timeout time.Duration) (func(ctx context.Context) error, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	port := strconv.Itoa(int(p.cfg.port))
	pg, err := container.Run(ctx, container.CreateRequest{
		Image: getPostGisImage(p.cfg.version),
		Env: map[string]string{
			"POSTGRES_PASSWORD": p.cfg.pass,
			"POSTGRES_USER":     p.cfg.user,
			"POSTGRES_DB":       p.cfg.name,
		},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		ExposedPorts: []string{fmt.Sprintf("%s:5432/tcp", port)},
		Name:         fmt.Sprintf("dbctl_pg_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{database.LabelType: database.LabelPostgres},
	})
	if err != nil {
		return nil, err
	}

	p.containerID = pg.ID

	closeFunc := func(ctx context.Context) error {
		return pg.Terminate(ctx)
	}

	return closeFunc, p.WaitForStart(ctx, timeout)
}

func (p *Postgres) URI() string {
	host := net.JoinHostPort("localhost", strconv.Itoa(int(p.cfg.port)))
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(p.cfg.user, p.cfg.pass), Host: host, Path: p.cfg.name, RawQuery: "sslmode=disable"}).String()
}

func RunMigrations(migrationsPath, uri string) error {
	if len(migrationsPath) == 0 {
		return nil
	}

	log.Println("Applying migrations ...")
	m, err := migrate.New(migrationsPath, uri)
	if err != nil {
		return fmt.Errorf("run migrations failed %w", err)
	}
	return m.Up()
}

func ApplyFixtures(ctx context.Context, fixtureFiles []string, uri string) error {
	if len(fixtureFiles) == 0 {
		return nil
	}

	log.Println("Applying fixtures ...")
	conn, err := pgx.Connect(ctx, uri)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	for _, f := range fixtureFiles {
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
