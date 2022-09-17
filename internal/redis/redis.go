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

	"github.com/gomodule/redigo/redis"
	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/pkg"
)

type Redis struct {
	cfg config
}

func New(options ...Option) (*Redis, error) {
	// create redis with default values
	pg := &Redis{cfg: config{
		pass:    "",
		user:    "",
		port:    16379,
		version: "7.0.4",
	}}

	for _, o := range options {
		if err := o(&pg.cfg); err != nil {
			return nil, err
		}
	}
	return pg, nil
}

func (p *Redis) Start() error {
	ctx := pkg.ContextWithOsSignal()
	log.Printf("Starting redis version %s on port %d ...\n", p.cfg.version, p.cfg.port)

	closeFunc, err := p.startUsingDocker(ctx)
	if err != nil {
		_ = closeFunc(ctx)
		return err
	}

	// print connection url
	log.Printf("Database uri is: %q\n", p.URI())

	// detach and stop cli if asked
	if p.cfg.detach {
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

func (p *Redis) startUsingDocker(ctx context.Context) (func(ctx context.Context) error, error) {
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
		Name:         fmt.Sprintf("dbctl_%d_%d", time.Now().Unix(), rnd.Uint64()),
	})
	if err != nil {
		return nil, err
	}

	closeFunc := func(ctx context.Context) error {
		return pg.Terminate(ctx)
	}

	uri := p.noAuthURI()
	if err := waitForRedis(ctx, uri); err != nil {
		return closeFunc, err
	}

	return closeFunc, p.setAuth(ctx, uri)
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

func waitForRedis(ctx context.Context, url string) error {
	log.Println("Wait for database to boot up")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	for range ticker.C {
		conn, err := redis.DialURLContext(ctx, url)
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
