package container

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Request struct {
	Name         string
	Image        string
	ExposedPorts []string // allow specifying protocol info
	Cmd          []string
	Env          map[string]string
}

type Container struct {
	ID string
}

func Run(ctx context.Context, req Request) (*Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	reader, err := cli.ImagePull(ctx, req.Image, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, reader)

	defer func() {
		_ = reader.Close()
	}()

	env := []string{}
	for envKey, envVar := range req.Env {
		env = append(env, envKey+"="+envVar)
	}

	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(req.ExposedPorts)
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          env,
		ExposedPorts: exposedPortSet,
	}, &container.HostConfig{PortBindings: exposedPortMap}, nil, nil, req.Name)
	if err != nil {
		return nil, err
	}

	cn := &Container{ID: resp.ID}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return cn, err
	}

	return cn, nil
}

func (c *Container) Terminate(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	})
	return err
}
