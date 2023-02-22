package container

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	LabelManagedBy = "managed_by"
	LabelDBctl     = "dbctl"
)

type Request struct {
	Name         string
	Image        string
	ExposedPorts []string // allow specifying protocol info
	Cmd          []string
	Env          map[string]string
	Labels       map[string]string
}

type Container struct {
	ID     string
	Name   string
	Labels map[string]string
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

	labels := map[string]string{LabelManagedBy: LabelDBctl}
	for k, v := range req.Labels {
		labels[k] = v
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          env,
		ExposedPorts: exposedPortSet,
		Labels:       labels,
	}, &container.HostConfig{PortBindings: exposedPortMap}, nil, nil, req.Name)
	if err != nil {
		return nil, err
	}

	cn := &Container{ID: resp.ID, Name: req.Name}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return cn, err
	}

	return cn, nil
}

func (c *Container) Terminate(ctx context.Context) error {
	return TerminateByID(ctx, c.ID)
}

func List(ctx context.Context, labels map[string]string) ([]*Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ff := []filters.KeyValuePair{{Key: "label", Value: LabelManagedBy + "=" + LabelDBctl}}
	for k, v := range labels {
		ff = append(ff, filters.Arg("label", k+"="+v))
	}

	res, err := cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(ff...),
	})
	if err != nil {
		return nil, err
	}

	out := make([]*Container, 0)
	for _, c := range res {
		out = append(out, &Container{ID: c.ID, Name: c.Names[0], Labels: c.Labels})
	}
	return out, nil
}

func TerminateByID(ctx context.Context, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	})
	return err
}
