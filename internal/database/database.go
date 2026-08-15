package database

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/dbctl/internal/container"
)

type Status int

type CloseFunc func(ctx context.Context) error

const (
	Running Status = iota
	Stoped
)

// Database type names accepted by the api server and its clients.
const (
	TypePostgres = "postgres"
	TypeRedis    = "redis"
	TypeMongoDB  = "mongodb"
)

const (
	LabelPostgres     = "postgres"
	LabelPGWeb        = "pgweb"
	LabelRedis        = "redis"
	LabelMongoDB      = "mongodb"
	LabelMongoExpress = "mongoexpress"
	LabelTesting      = "testing"
)

// InstanceLabels builds the container label filter for a database type, narrowed
// down to a user supplied label when one is given. Without it, commands would act
// on every dbctl instance on the machine, including the ones another project is
// running.
func InstanceLabels(dbType, label string) map[string]string {
	labels := map[string]string{container.LabelType: dbType}
	if label != "" {
		labels[container.LabelCustom] = label
	}
	return labels
}

// ConnectionLabels records how to reach an instance on the container running it,
// so that it can be found again without knowing which port it was started on.
func ConnectionLabels(user, pass, name string, port uint32) map[string]string {
	return map[string]string{
		container.LabelPort: strconv.FormatUint(uint64(port), 10),
		container.LabelUser: user,
		container.LabelPass: pass,
		container.LabelName: name,
	}
}

// Instance is a running database instance and the details needed to connect to it.
type Instance struct {
	ID    string
	Type  string
	Label string

	Port uint32
	User string
	Pass string
	Name string
}

// ErrNoInstance is returned when no running instance of a type can be found.
var ErrNoInstance = errors.New("no running instance found")

// ErrAmbiguousInstance is returned when several instances of a type are running and
// no label picks one of them.
var ErrAmbiguousInstance = errors.New("more than one running instance")

// FindInstance locates the running instance of the given type, so callers do not
// have to know the port it was started on. When more than one is running the label
// is what tells them apart.
func FindInstance(ctx context.Context, dbType, label string) (*Instance, error) {
	found, err := container.List(ctx, InstanceLabels(dbType, label))
	if err != nil {
		return nil, err
	}

	if len(found) == 0 {
		if label != "" {
			return nil, fmt.Errorf("%w: no %s instance labelled %q, start one with 'dbctl start %s --label %s'",
				ErrNoInstance, dbType, label, dbType, label)
		}
		return nil, fmt.Errorf("%w: no %s instance, start one with 'dbctl start %s'", ErrNoInstance, dbType, dbType)
	}

	if len(found) > 1 {
		names := make([]string, 0, len(found))
		for _, c := range found {
			names = append(names, fmt.Sprintf("%s (label %q)", c.Name, c.Labels[container.LabelCustom]))
		}
		return nil, fmt.Errorf("%w: %d %s instances are running, pass --label to pick one: %s",
			ErrAmbiguousInstance, len(found), dbType, strings.Join(names, ", "))
	}

	c := found[0]
	out := &Instance{
		ID:    c.ID,
		Type:  dbType,
		Label: c.Labels[container.LabelCustom],
		User:  c.Labels[container.LabelUser],
		Pass:  c.Labels[container.LabelPass],
		Name:  c.Labels[container.LabelName],
	}

	// instances started by an older dbctl carry no connection labels, the caller
	// falls back to the defaults of the type in that case.
	if p := c.Labels[container.LabelPort]; p != "" {
		port, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("instance %s has an invalid port label %q: %w", c.Name, p, err)
		}
		out.Port = uint32(port)
	}

	return out, nil
}

// FindInstances locates every running instance carrying the given label, keyed by
// database type. Types that are not running, or that have several instances and no
// label to tell them apart, are left out.
func FindInstances(ctx context.Context, label string) (map[string]*Instance, error) {
	out := make(map[string]*Instance)
	for _, dbType := range []string{TypePostgres, TypeRedis, TypeMongoDB} {
		instance, err := FindInstance(ctx, dbType, label)
		if err != nil {
			if errors.Is(err, ErrNoInstance) || errors.Is(err, ErrAmbiguousInstance) {
				continue
			}
			return nil, err
		}
		out[dbType] = instance
	}
	return out, nil
}

type Info struct {
	ID     string
	Type   string
	Status Status
}

type Database interface {
	Start(ctx context.Context, detach bool) error
	Stop(ctx context.Context) error
	WaitForStart(ctx context.Context, timeout time.Duration) error
	URI() string
}

type CreateDBRequest struct {
	Migrations string
	Fixtures   string

	WithDefaultMigrations bool
}

type CreateDBResponse struct {
	URI string
}

type Admin interface {
	CreateDB(ctx context.Context, req *CreateDBRequest) (*CreateDBResponse, error)
	RemoveDB(ctx context.Context, uri string) error
}
