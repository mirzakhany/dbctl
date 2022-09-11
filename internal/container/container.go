package container

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
)

func Run(ctx context.Context, req testcontainers.ContainerRequest) (testcontainers.Container, error) {
	cn, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	return cn, nil
}
