package redis

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/logger"
	"github.com/mirzakhany/dbctl/internal/utils"
)

var (
	_ database.Database = (*Redis)(nil)
	_ database.Admin    = (*Redis)(nil)
)

const (
	// defaultDatabaseCount is the number of databases a redis instance exposes
	// unless it is configured otherwise
	defaultDatabaseCount = 16
	// DefaultPort is the default port for redis
	DefaultPort = 16379
	// DefaultUser is the default user for redis
	DefaultUser = ""
	// DefaultPass is the default password for redis
	DefaultPass = ""
)

// Redis is a redis database
type Redis struct {
	containerID string
	cfg         config
}

// New creates a new redis database instance
func New(options ...Option) (*Redis, error) {
	// create redis with default values
	rs := &Redis{cfg: config{
		pass:    DefaultPass,
		user:    DefaultUser,
		port:    DefaultPort,
		version: "7.0.4",
	}}

	for _, o := range options {
		if err := o(&rs.cfg); err != nil {
			return nil, err
		}
	}

	return rs, nil
}

// indexKeyPrefix marks a redis database index as handed out. The keys live in
// index 0, which dbctl keeps for its own bookkeeping and never hands out.
const indexKeyPrefix = "dbctl:db:"

// CreateDB creates a new database
func (p *Redis) CreateDB(ctx context.Context, req *database.CreateDBRequest) (*database.CreateDBResponse, error) {
	conn, err := redis.DialURLContext(ctx, p.adminURI())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	// get first available db index
	dbIndex, err := p.acquireDBIndex(conn)
	if err != nil {
		return nil, err
	}

	newDB, err := New(WithHost(p.cfg.user, p.cfg.pass, dbIndex, p.cfg.port))
	if err != nil {
		return nil, err
	}

	if req != nil && req.Fixtures != "" {
		files, err := getFiles(req.Fixtures)
		if err != nil {
			return nil, fmt.Errorf("read fixtures failed: %w", err)
		}

		if err := p.applyFixturesTo(ctx, newDB.URI(), files); err != nil {
			return nil, err
		}
	}

	return &database.CreateDBResponse{URI: hostURI(newDB.URI())}, nil
}

// applyFixturesTo loads the fixture files into the database the uri points at.
func (p *Redis) applyFixturesTo(ctx context.Context, uri string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	conn, err := redis.DialURLContext(ctx, uri)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	return applyFixtures(conn, files)
}

// acquireDBIndex claims the first unused redis database index. Redis exposes a
// fixed set of numbered databases (16 by default) rather than named ones, so the
// claim has to be recorded to stop two callers from sharing one index.
func (p *Redis) acquireDBIndex(conn redis.Conn) (int, error) {
	count, err := databaseCount(conn)
	if err != nil {
		return 0, err
	}

	// index 0 holds the bookkeeping keys, indexes 1..count-1 are handed out.
	stmt := redis.NewScript(0, `
		local count = tonumber(ARGV[1])
		local prefix = ARGV[2]
		for index = 1, count - 1 do
			if redis.call("SETNX", prefix .. index, 1) == 1 then
				redis.call("SELECT", index)
				redis.call("FLUSHDB")
				redis.call("SELECT", 0)
				return index
			end
		end
		return -1
	`)

	index, err := redis.Int(stmt.Do(conn, count, indexKeyPrefix))
	if err != nil {
		return 0, err
	}

	if index < 0 {
		return 0, fmt.Errorf("no free redis database left, all %d indexes of this instance are in use, "+
			"remove the databases you no longer need", count-1)
	}

	return index, nil
}

// databaseCount reports how many databases this redis instance exposes.
func databaseCount(conn redis.Conn) (int, error) {
	values, err := redis.Strings(conn.Do("CONFIG", "GET", "databases"))
	if err != nil || len(values) != 2 {
		// CONFIG may be renamed or disabled, fall back to the redis default.
		return defaultDatabaseCount, nil //nolint:nilerr
	}

	count, err := strconv.Atoi(values[1])
	if err != nil || count < 2 {
		return defaultDatabaseCount, nil //nolint:nilerr
	}

	return count, nil
}

// RemoveDB removes a database by its uri
func (p *Redis) RemoveDB(ctx context.Context, uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// get db index from uri
	dbIndex, err := strconv.Atoi(strings.TrimPrefix(u.Path, "/"))
	if err != nil {
		return fmt.Errorf("uri %q does not contain a database index: %w", uri, err)
	}

	// index 0 is dbctl's own bookkeeping database, it is never handed out and
	// must not be flushed.
	if dbIndex == 0 {
		return nil
	}

	// flush the database and release the index for reuse
	stmt := redis.NewScript(0, `
		local dbIndex = tonumber(ARGV[1])
		redis.call("SELECT", dbIndex)
		redis.call("FLUSHDB")
		redis.call("SELECT", 0)
		redis.call("DEL", ARGV[2] .. dbIndex)
	`)

	conn, err := redis.DialURLContext(ctx, p.adminURI())
	if err != nil {
		return err
	}

	defer func() {
		_ = conn.Close()
	}()

	if _, err := stmt.Do(conn, dbIndex, indexKeyPrefix); err != nil && !errors.Is(err, redis.ErrNil) {
		return err
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

// Start starts the database
func (p *Redis) Start(ctx context.Context, detach bool) error {
	log.Printf("Starting redis version %s ...\n", p.cfg.version)

	closeFunc, err := p.startUsingDocker(ctx, 20*time.Second)
	if err != nil {
		_ = closeFunc(ctx)
		return err
	}

	// apply fixtures to the database this instance points at
	if err := p.applyFixturesTo(ctx, p.URI(), p.cfg.fixtureFiles); err != nil {
		_ = closeFunc(ctx)
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
	logger.Info("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	return closeFunc(shutdownCtx)
}

// Stop stops the database
func (p *Redis) Stop(ctx context.Context) error {
	return container.TerminateByID(ctx, p.containerID)
}

// WaitForStart waits for database to boot up
func (p *Redis) WaitForStart(ctx context.Context, timeout time.Duration) error {
	logger.Info("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for range ticker.C {
		conn, err := redis.DialURLContext(ctx, p.noAuthURI())
		if err != nil {
			if err == context.DeadlineExceeded {
				return err
			}
		} else {
			_ = conn.Close()
			return nil
		}
	}
	return nil
}

// Instances returns a list of running redis instances
// Instances returns the running redis instances, restricted to the given label
// when one is provided.
func Instances(ctx context.Context, label string) ([]database.Info, error) {
	l, err := container.List(ctx, database.InstanceLabels(database.LabelRedis, label))
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

func (p *Redis) startUsingDocker(ctx context.Context, timeout time.Duration) (func(ctx context.Context) error, error) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return nil, err
	}

	// port 0 asks for any free port, so that several projects can run their tests
	// at the same time.
	if p.cfg.port == 0 {
		p.cfg.port = uint32(utils.GetAvailablePort())
	}

	port := strconv.Itoa(int(p.cfg.port))

	req := container.CreateRequest{
		Image: getRedisImage(p.cfg.version),
		Cmd: []string{
			"redis-server",
			"--save", "",
			"--databases", "2000",
		},
		ExposedPorts: []string{container.PortSpec(port, "6379/tcp")},
		Name:         fmt.Sprintf("dbctl_rs_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelRedis},
	}

	for k, v := range database.ConnectionLabels(p.cfg.user, p.cfg.pass, "", p.cfg.port) {
		req.Labels[k] = v
	}

	if p.cfg.label != "" {
		req.Labels[container.LabelCustom] = p.cfg.label
	}

	pg, err := container.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	closeFunc := func(ctx context.Context) error {
		return pg.Terminate(ctx)
	}

	if err := p.WaitForStart(ctx, timeout); err != nil {
		return nil, err
	}

	return closeFunc, p.setAuth(ctx, p.noAuthURI())
}

// noAuthURI points at index 0 without credentials. It is only valid while the
// instance is booting, before setAuth has configured a password.
func (p *Redis) noAuthURI() string {
	addr := "localhost"
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		addr = "host.docker.internal"
	}

	return (&url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort(addr, strconv.Itoa(int(p.cfg.port))),
		Path:   "0",
	}).String()
}

// adminURI is the connection string dbctl uses for its own bookkeeping. It always
// points at index 0 but, unlike noAuthURI, it keeps the credentials so that
// instances started with a password still work.
func (p *Redis) adminURI() string {
	admin, err := New(WithHost(p.cfg.user, p.cfg.pass, 0, p.cfg.port))
	if err != nil {
		// the options used here never fail, fall back to this instance's uri.
		return p.URI()
	}
	return admin.URI()
}

// URI returns the connection string for the database
func (p *Redis) URI() string {
	addr := "localhost"
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		addr = "host.docker.internal"
	}

	host := net.JoinHostPort(addr, strconv.Itoa(int(p.cfg.port)))

	var userInfo *url.Userinfo
	if p.cfg.user != "" && p.cfg.pass != "" {
		userInfo = url.UserPassword(p.cfg.user, p.cfg.pass)
	} else if p.cfg.user != "" {
		userInfo = url.User(p.cfg.user)
	}

	return (&url.URL{
		Scheme: "redis",
		User:   userInfo,
		Host:   host,
		Path:   strconv.Itoa(p.cfg.dbIndex),
	}).String()
}

func (p *Redis) setAuth(ctx context.Context, url string) error {
	if p.cfg.user == "" && p.cfg.pass == "" {
		return nil
	}

	conn, err := redis.DialURLContext(ctx, url)
	if err != nil {
		return err
	}

	args := []interface{}{}
	args = append(args, "SETUSER", p.cfg.user)
	if p.cfg.pass != "" {
		args = append(args, "on", fmt.Sprintf(">%s", p.cfg.pass))
	}
	args = append(args, "~*", "&*", "+@all")

	_, err = redis.DoContext(conn, ctx, "ACL", args...)
	return err
}
