package pg

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/dbctl/internal/logger"

	// golang postgres driver
	_ "github.com/lib/pq"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
)

var (
	_ database.Database = (*Postgres)(nil)
	_ database.Admin    = (*Postgres)(nil)

	errDatabaseNotExists = errors.New("database does not exist")
)

const (
	// DefaultPort is the default port for postgres
	DefaultPort = 15432
	// DefaultUser is the default user for postgres
	DefaultUser = "postgres"
	// DefaultPass is the default password for postgres
	DefaultPass = "postgres"
	// DefaultName is the default database name for postgres
	DefaultName = "postgres"
	// DefaultTemplate is the default template name for postgres when creating a new database with migtations and fixtures
	DefaultTemplate = "dbctl_template"
	// DefaultVersion is the postgres version started when none is requested
	DefaultVersion = "14.3.2"
)

// Postgres is a postgres database instance
type Postgres struct {
	containerID string
	cfg         config
}

// New creates a new postgres database instance controller
func New(options ...Option) (*Postgres, error) {
	// create postgres with default values
	pg := &Postgres{cfg: config{
		pass:    DefaultPass,
		user:    DefaultUser,
		name:    DefaultName,
		port:    DefaultPort,
		version: DefaultVersion,
	}}

	for _, o := range options {
		if err := o(&pg.cfg); err != nil {
			return nil, err
		}
	}

	return pg, nil
}

// CreateDB creates a new database with given migrations and fixtures
func (p *Postgres) CreateDB(ctx context.Context, req *database.CreateDBRequest) (*database.CreateDBResponse, error) {
	// connect to default database
	conn, err := dbConnect(ctx, p.URI())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	// create a random name for new database
	dbName := fmt.Sprintf("dbctl_%d", time.Now().UnixNano())
	newDB, err := New(WithHost(p.cfg.user, p.cfg.pass, dbName, p.cfg.port))
	if err != nil {
		return nil, err
	}
	newURI := newDB.URI()

	switch {
	case req.WithDefaultMigrations:
		if err := p.createDatabaseWithTemplate(ctx, conn, dbName, DefaultTemplate); err != nil {
			if errors.Is(err, errDatabaseNotExists) {
				return nil, fmt.Errorf("default template database %q not found, start the instance with migrations first: %w", DefaultTemplate, err)
			}
			return nil, err
		}

	case len(req.Migrations) == 0:
		logger.Debug("No migrations provided, creating a new database ...")
		if err := createDatabase(ctx, conn, dbName); err != nil {
			return nil, err
		}

	default:
		if err := p.createDatabaseFromMigrations(ctx, conn, dbName, newURI, req.Migrations); err != nil {
			return nil, err
		}
	}

	// fixtures always go to the database that was just created, never to the
	// maintenance connection this method holds.
	if len(req.Fixtures) != 0 {
		if err := applyFixturesFromDir(ctx, nil, req.Fixtures, newURI); err != nil {
			return nil, err
		}
	}

	return &database.CreateDBResponse{URI: hostURI(newURI)}, nil
}

// createDatabaseFromMigrations creates dbName from a template holding the given
// migrations, building that template first if this is the first time these
// migrations are seen.
func (p *Postgres) createDatabaseFromMigrations(ctx context.Context, conn *sql.DB, dbName, dbURI, migrationsPath string) error {
	logger.Debug("Creating a new database with migrations ...")

	files, err := getFiles(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migraions failed: %w", err)
	}

	migrationFiles := MigrationFiles(files)
	if len(migrationFiles) == 0 {
		logger.Debug("No migration files found, creating a new database ...")
		return createDatabase(ctx, conn, dbName)
	}

	template, err := templateName(migrationFiles)
	if err != nil {
		return err
	}
	logger.Debug("template name is:", template)

	// try to create the database from the template built by an earlier request
	err = p.createDatabaseWithTemplate(ctx, conn, dbName, template)
	if err == nil {
		return nil
	}
	if !errors.Is(err, errDatabaseNotExists) {
		return err
	}

	logger.Debug("template database not found, creating a new database ...")
	if err := createDatabase(ctx, conn, dbName); err != nil {
		return err
	}

	// connect to new database and run migrations
	if err := RunMigrations(ctx, nil, migrationFiles, dbURI); err != nil {
		return err
	}

	// snapshot it as the template for the next request. a concurrent request may
	// have created it in the meantime, which is harmless.
	if err := p.createDatabaseWithTemplate(ctx, conn, template, dbName); err != nil {
		logger.Debug("creating template database failed, migrations will run again next time:", err)
	}

	return nil
}

// hostURI rewrites a uri so that it is usable by the caller. The api server reaches
// the databases through host.docker.internal, its clients run outside docker.
func hostURI(uri string) string {
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		return strings.ReplaceAll(uri, "host.docker.internal", "localhost")
	}
	return uri
}

func (p *Postgres) createDatabaseWithTemplate(ctx context.Context, conn *sql.DB, name, template string) error {
	if conn == nil {
		var err error
		conn, err = dbConnect(ctx, p.URI())
		if err != nil {
			return err
		}
		defer func() {
			_ = conn.Close()
		}()
	}

	// if default is exist, use it as template and create new database
	if _, err := conn.Exec(fmt.Sprintf("create database %s with template %s",
		quoteIdentifier(name), quoteIdentifier(template))); err != nil {
		// is error database not exist?
		if strings.Contains(err.Error(), "does not exist") {
			return errDatabaseNotExists
		}
		return fmt.Errorf("create database with template failed: %w", err)
	}
	return nil
}

// RemoveDB removes a database from postgres by given uri
func (p *Postgres) RemoveDB(ctx context.Context, uri string) error {
	// parse the uri to get database name
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// get database name
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return fmt.Errorf("uri %q does not contain a database name", uri)
	}

	conn, err := dbConnect(ctx, p.URI())
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	// terminate the sessions still connected to it, postgres refuses to drop a
	// database while anyone is attached to it and callers commonly still hold a
	// pooled connection at cleanup time.
	if _, err := conn.ExecContext(ctx,
		"select pg_terminate_backend(pid) from pg_stat_activity where datname = $1 and pid <> pg_backend_pid()",
		dbName); err != nil {
		return fmt.Errorf("terminating connections to database %q failed: %w", dbName, err)
	}

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("drop database if exists %s", quoteIdentifier(dbName))); err != nil {
		return fmt.Errorf("drop database failed: %w", err)
	}

	return nil
}

// quoteIdentifier quotes a postgres identifier so that generated names are never
// interpreted as syntax.
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Start starts a postgres database
func (p *Postgres) Start(ctx context.Context, detach bool) error {
	logger.Info(fmt.Sprintf("Starting postgres version %s on port %d ...", p.cfg.version, p.cfg.port))

	closeFunc, err := p.startUsingDocker(ctx, 20*time.Second)
	if err != nil {
		return err
	}

	logger.Info("Postgres is up and running")
	// run migrations if exist
	if err := RunMigrations(ctx, nil, p.cfg.migrationsFiles, p.URI()); err != nil {
		return err
	}

	// create template database if migrations exist
	if len(p.cfg.migrationsFiles) > 0 {
		_ = p.createDatabaseWithTemplate(ctx, nil, DefaultTemplate, p.cfg.name)

		// run apply fixtures if exist
		if err := ApplyFixtures(ctx, nil, p.cfg.fixtureFiles, p.URI()); err != nil {
			return err
		}
	}

	// print connection url
	logger.Info(fmt.Sprintf("Database uri is: %q", p.URI()))

	var pgwebCloseFunc database.CloseFunc
	if p.cfg.withUI {
		pgwebCloseFunc, err = p.runUI(ctx)
		if err != nil {
			_ = closeFunc(ctx)
			return err
		}
	}

	// detach and stop cli if asked
	if detach {
		return nil
	}

	<-ctx.Done()
	logger.Info("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	// TODO we need a better solution to manage containers and make sure we remove all of them.
	if pgwebCloseFunc != nil {
		if err := pgwebCloseFunc(shutdownCtx); err != nil {
			return err
		}
	}

	return closeFunc(shutdownCtx)
}

// Stop stops a postgres database
func (p *Postgres) Stop(ctx context.Context) error {
	return container.TerminateByID(ctx, p.containerID)
}

// WaitForStart waits for postgres to start
func (p *Postgres) WaitForStart(ctx context.Context, timeout time.Duration) error {
	logger.Info("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for range ticker.C {
		conn, err := dbConnect(ctx, p.URI())
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return err
			}
		} else {
			_ = conn.Close()
			return nil
		}
	}
	return nil
}

func (p *Postgres) runUI(ctx context.Context) (database.CloseFunc, error) {
	logger.Info("Starting postgres ui using pgweb (https://github.com/sosedoff/pgweb)")

	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	pgweb, err := container.Run(ctx, container.CreateRequest{
		Image: "sosedoff/pgweb:latest",
		Env: map[string]string{
			// replace localhost with docker internal network
			"PGWEB_DATABASE_URL": strings.ReplaceAll(p.URI(), "localhost", "host.docker.internal"),
		},
		ExposedPorts: []string{"8081:8081"},
		Name:         fmt.Sprintf("dbctl_pgweb_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelPGWeb},
	})
	if err != nil {
		return nil, err
	}

	// log ui url
	logger.Info("Database UI is running on: http://localhost:8081")

	closeFunc := func(ctx context.Context) error {
		return pgweb.Terminate(ctx)
	}

	return closeFunc, nil
}

// Instances returns a list of postgres instances
// Instances returns the running postgres instances, restricted to the given label
// when one is provided.
func Instances(ctx context.Context, label string) ([]database.Info, error) {
	l, err := container.List(ctx, database.InstanceLabels(database.LabelPostgres, label))
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

func (p *Postgres) startUsingDocker(ctx context.Context, timeout time.Duration) (database.CloseFunc, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	port := strconv.Itoa(int(p.cfg.port))
	req := container.CreateRequest{
		Image: getPostGisImage(p.cfg.version),
		Env: map[string]string{
			"POSTGRES_PASSWORD": p.cfg.pass,
			"POSTGRES_USER":     p.cfg.user,
			"POSTGRES_DB":       p.cfg.name,
		},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		ExposedPorts: []string{fmt.Sprintf("%s:5432/tcp", port)},
		Name:         fmt.Sprintf("dbctl_pg_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelPostgres},
	}

	if p.cfg.label != "" {
		req.Labels[container.LabelCustom] = p.cfg.label
	}

	pg, err := container.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	p.containerID = pg.ID

	closeFunc := func(ctx context.Context) error {
		return pg.Terminate(ctx)
	}

	return closeFunc, p.WaitForStart(ctx, timeout)
}

// URI returns the postgres connection uri
func (p *Postgres) URI() string {
	addr := "localhost"
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		addr = "host.docker.internal"
	}

	host := net.JoinHostPort(addr, strconv.Itoa(int(p.cfg.port)))
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(p.cfg.user, p.cfg.pass), Host: host, Path: p.cfg.name, RawQuery: "sslmode=disable"}).String()
}

func (p *Postgres) ContainerID() string {
	return p.containerID
}

// RunMigrations runs migrations on a postgres database
func RunMigrations(ctx context.Context, conn *sql.DB, migrationsFiles []string, uri string) error {
	if migrationsFiles == nil {
		return nil
	}

	logger.Info("Applying migrations ...")
	return applySQL(ctx, conn, migrationsFiles, uri)
}

// ApplyFixtures applies fixtures on a postgres database
func ApplyFixtures(ctx context.Context, conn *sql.DB, fixtureFiles []string, uri string) error {
	if len(fixtureFiles) == 0 {
		return nil
	}

	logger.Info("Applying fixtures ...")
	return applySQL(ctx, conn, fixtureFiles, uri)
}

func applyFixturesFromDir(ctx context.Context, conn *sql.DB, dir string, uri string) error {
	if dir == "" {
		return nil
	}

	files, err := getFiles(dir)
	if err != nil {
		return fmt.Errorf("read fixtures failed: %w", err)
	}

	logger.Info("Applying fixtures ...")
	return applySQL(ctx, conn, files, uri)
}

func createDatabase(ctx context.Context, conn *sql.DB, name string) error {
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("create database %s", quoteIdentifier(name))); err != nil {
		return fmt.Errorf("create database failed: %w", err)
	}
	return nil
}

func applySQL(ctx context.Context, conn *sql.DB, stmts []string, uri string) error {
	if conn == nil {
		var err error
		conn, err = dbConnect(ctx, uri)
		if err != nil {
			return err
		}
		defer func() {
			_ = conn.Close()
		}()
	}

	for _, f := range stmts {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read file (%s) failed: %w", f, err)
		}

		// one transaction per file, so a failing file does not leave a half
		// applied schema behind that would then be cached as a template.
		if err := applyInTx(ctx, conn, string(b)); err != nil {
			return fmt.Errorf("applying file (%s) failed: %w", f, err)
		}
	}
	return nil
}

func applyInTx(ctx context.Context, conn *sql.DB, stmt string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func dbConnect(ctx context.Context, uri string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}
