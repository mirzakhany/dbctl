package database

import (
	"time"

	"golang.org/x/net/context"
)

type Status int

const (
	Running Status = iota
	Stoped
)

const (
	LabelManagedBy = "managed_by"
	LabelType      = "dbctl_type"
	LabelDBctl     = "dbctl"
	LabelPostgres  = "postgres"
	LabelRedis     = "redis"
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
	// Instances(ctx context.Context) ([]Info, error)
	URI() string
}
