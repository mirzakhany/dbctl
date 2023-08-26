package database

import (
	"context"
	"time"
)

type Status int

type CloseFunc func(ctx context.Context) error

const (
	Running Status = iota
	Stoped
)

const (
	LabelType     = "dbctl_type"
	LabelPostgres = "postgres"
	LabelPGWeb    = "pgweb"
	LabelRedis    = "redis"
	LabelTesting  = "testing"
)

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
}

type CreateDBResponse struct {
	URI string
}

type Admin interface {
	CreateDB(ctx context.Context, req *CreateDBRequest) (*CreateDBResponse, error)
	RemoveDB(ctx context.Context, uri string) error
}
