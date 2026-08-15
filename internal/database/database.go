package database

import (
	"context"
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
