package database

import (
	"context"
	"time"
)

type Status int

const (
	Running Status = iota
	Stoped
)

const (
	LabelType     = "dbctl_type"
	LabelPostgres = "postgres"
	LabelRedis    = "redis"
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
