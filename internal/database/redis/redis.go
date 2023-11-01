package redis

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
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
	"github.com/mirzakhany/dbctl/internal/logger"
)

var (
	_ database.Database = (*Redis)(nil)
	_ database.Admin    = (*Redis)(nil)
)

const (
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

// CreateDB creates a new database
func (p *Redis) CreateDB(ctx context.Context, req *database.CreateDBRequest) (*database.CreateDBResponse, error) {
	// get first available db index
	dbIndex, err := p.getAvailableDBIndex(ctx)
	if err != nil {
		return nil, err
	}

	p.cfg.dbIndex = dbIndex
	uri := p.URI()
	// make sure we retrun localhost instead of host.docker.internal
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		uri = strings.ReplaceAll(uri, "host.docker.internal", "localhost")
	}

	return &database.CreateDBResponse{URI: uri}, nil
}

func (p *Redis) getAvailableDBIndex(ctx context.Context) (int, error) {
	// get or saw db index
	conn, err := redis.DialURLContext(ctx, p.noAuthURI())
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = conn.Close()
	}()

	stmt := redis.NewScript(0, `
		local dbIndex = redis.call("GET", "dbctl:dbIndex")
		if not dbIndex then
			dbIndex = 1
		end
		redis.call("SET", "dbctl:dbIndex", dbIndex+1)
		redis.call("SELECT", dbIndex)
		redis.call("FLUSHDB")
		return dbIndex
	`)

	return redis.Int(stmt.Do(conn))
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
		return err
	}

	// remove db index and flush db
	stmt := redis.NewScript(0, `
		local dbIndex = tonumber(ARGV[1])
		if dbIndex == 0 then
			return
		end
		redis.call("SELECT", dbIndex)
		redis.call("FLUSHDB")
		redis.call("SET", "dbctl:dbIndex", dbIndex-1)
	`)

	conn, err := redis.DialURLContext(ctx, p.noAuthURI())
	if err != nil {
		return err
	}

	defer func() {
		_ = conn.Close()
	}()

	_, err = stmt.Do(conn, dbIndex)
	return err
}

// Start starts the database
func (p *Redis) Start(ctx context.Context, detach bool) error {
	log.Printf("Starting redis version %s on port %d ...\n", p.cfg.version, p.cfg.port)

	closeFunc, err := p.startUsingDocker(ctx, 20*time.Second)
	if err != nil {
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
func Instances(ctx context.Context) ([]database.Info, error) {
	l, err := container.List(ctx, map[string]string{container.LabelType: database.LabelRedis})
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

	port := strconv.Itoa(int(p.cfg.port))

	req := container.CreateRequest{
		Image: getRedisImage(p.cfg.version),
		Cmd: []string{
			"redis-server",
			"--save", "",
			"--databases", "2000",
		},
		ExposedPorts: []string{fmt.Sprintf("%s:6379/tcp", port)},
		Name:         fmt.Sprintf("dbctl_rs_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: database.LabelRedis},
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

func (p *Redis) noAuthURI() string {
	addr := "localhost"
	if os.Getenv("DBCTL_INSIDE_DOCKER") == "true" {
		addr = "host.docker.internal"
	}

	return (&url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort(addr, strconv.Itoa(int(p.cfg.port))),
		Path:   strconv.Itoa(p.cfg.dbIndex),
	}).String()
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
