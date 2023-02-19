package redis

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/gomodule/redigo/redis"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/database"
)

var _ database.Database = (*Redis)(nil)

type Redis struct {
	containerID string
	cfg         config
}

func New(options ...Option) (*Redis, error) {
	// create redis with default values
	rs := &Redis{cfg: config{
		pass:    "",
		user:    "",
		port:    16379,
		version: "7.0.4",
	}}

	for _, o := range options {
		if err := o(&rs.cfg); err != nil {
			return nil, err
		}
	}
	return rs, nil
}

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
	log.Println("Shutdown signal received, stopping database")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	return closeFunc(shutdownCtx)
}

func (p *Redis) Stop(ctx context.Context) error {
	return container.TerminateByID(ctx, p.containerID)
}

func (p *Redis) WaitForStart(ctx context.Context, timeout time.Duration) error {
	log.Println("Wait for database to boot up")
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

func Instances(ctx context.Context) ([]database.Info, error) {
	l, err := container.List(ctx, filters.KeyValuePair{Key: database.LabelType, Value: database.LabelRedis})
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
	pg, err := container.Run(ctx, container.Request{
		Image: getRedisImage(p.cfg.version),
		Cmd: []string{
			"redis-server",
			"--save", "",
			"--databases", "2000",
		},
		ExposedPorts: []string{fmt.Sprintf("%s:6379/tcp", port)},
		Name:         fmt.Sprintf("dbctl_rs_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{database.LabelType: database.LabelRedis},
	})
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
	return (&url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort("localhost", strconv.Itoa(int(p.cfg.port))),
		Path:   strconv.Itoa(p.cfg.dbIndex),
	}).String()
}

func (p *Redis) URI() string {
	host := net.JoinHostPort("localhost", strconv.Itoa(int(p.cfg.port)))

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