package container

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/mirzakhany/dbctl/internal/logger"
)

const (
	// LabelManagedBy is the label used to identify containers managed by dbctl
	LabelManagedBy = "managed_by"
	// LabelDBctl is the value of the managed_by label
	LabelDBctl = "dbctl"
)

// Run creates and starts a container
func Run(ctx context.Context, req CreateRequest) (*Container, error) {
	if err := PullImage(ctx, req.Image); err != nil {
		return nil, err
	}

	id, err := CreateContainer(ctx, CreateRequest{
		Name:         req.Name,
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          req.Env,
		ExposedPorts: req.ExposedPorts,
		Labels:       req.Labels,
	})
	if err != nil {
		return nil, err
	}

	cn := &Container{ID: id, Name: req.Name}
	if err := StartContainer(ctx, id); err != nil {
		return cn, err
	}

	return cn, nil
}

// Terminate stops and removes a container
func (c *Container) Terminate(ctx context.Context) error {
	return TerminateByID(ctx, c.ID)
}

// TerminateByID stops and removes a container by id
func TerminateByID(ctx context.Context, id string) error {
	return RemoveContainer(ctx, id)
}

// StartContainer starts a container by id
func StartContainer(ctx context.Context, id string) error {
	cl, closer, err := getDockerClient()
	if err != nil {
		return err
	}
	defer closer()

	return cl.ContainerStart(ctx, id, container.StartOptions{})
}

// CreateContainer creates a container
func CreateContainer(ctx context.Context, params CreateRequest) (string, error) {
	cl, closer, err := getDockerClient()
	if err != nil {
		return "", err
	}
	defer closer()

	labels := map[string]string{LabelManagedBy: LabelDBctl}
	for k, v := range params.Labels {
		labels[k] = v
	}

	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(params.ExposedPorts)
	if err != nil {
		return "", err
	}

	for _, pm := range exposedPortMap {
		for _, port := range pm {
			if !isPortFree(port.HostPort) {
				return "", fmt.Errorf("port: '%s' is already taken", port.HostPort)
			}
		}
	}

	envs := make([]string, 0, len(params.Env))
	for k, v := range params.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	resp, err := cl.ContainerCreate(ctx, &container.Config{
		Image:        params.Image,
		Cmd:          params.Cmd,
		Env:          envs,
		Labels:       labels,
		ExposedPorts: exposedPortSet,
	}, &container.HostConfig{
		PortBindings: exposedPortMap,
	}, nil,
		nil,
		params.Name,
	)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// PullImage pulls a docker image
func PullImage(ctx context.Context, image string) error {
	logger.Info(fmt.Sprintf("Pulling docker image: %q, depends on your connection speed it might take upto minutes", image))
	cl, closer, err := getDockerClient()
	if err != nil {
		return err
	}
	defer closer()

	res, err := cl.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	// read the body to make sure we wait for image to get pulled
	_, err = io.ReadAll(res)
	return err
}

// List lists all containers with the given labels managed by dbctl
func List(ctx context.Context, labels map[string]string) ([]*Container, error) {
	cl, closer, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	defer closer()

	kvPairs := []filters.KeyValuePair{{Key: "label", Value: LabelManagedBy + "=" + LabelDBctl}}
	for k, v := range labels {
		kvPairs = append(kvPairs, filters.Arg("label", k+"="+v))
	}

	containers, err := cl.ContainerList(ctx, container.ListOptions{
		Limit:   0,
		Filters: filters.NewArgs(kvPairs...),
	})
	if err != nil {
		return nil, err
	}

	out := make([]*Container, 0, len(containers))
	for _, c := range containers {
		out = append(out, &Container{
			ID:     c.ID,
			Name:   c.Names[0],
			Labels: c.Labels,
		})
	}

	return out, nil
}

// RemoveContainer removes a container by id
func RemoveContainer(ctx context.Context, id string) error {
	cl, closer, err := getDockerClient()
	if err != nil {
		return err
	}
	defer closer()

	return cl.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		Force:         true,
		RemoveLinks:   false,
		RemoveVolumes: true,
	})
}

type DockerExecResponse struct {
	ID string `json:"Id"`
}

func CreateExec(ctx context.Context, containerID string, cmd []string) (string, error) {
	cl, closer, err := getDockerClient()
	if err != nil {
		return "", err
	}
	defer closer()

	resp, err := cl.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Cmd:          cmd,
		Detach:       false,
		Tty:          false,
		AttachStdout: true,
		AttachStdin:  false,
		AttachStderr: true,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func StartExec(ctx context.Context, execID string) (string, error) {
	cl, closer, err := getDockerClient()
	if err != nil {
		return "", err
	}
	defer closer()

	resp, err := cl.ContainerExecAttach(ctx, execID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		return "", err
	}
	defer resp.Close()

	body, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func RunExec(ctx context.Context, containerID string, cmd []string) (string, error) {
	execID, err := CreateExec(ctx, containerID, cmd)
	if err != nil {
		return "", err
	}

	return StartExec(ctx, execID)
}

func isPortFree(port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), 3*time.Second)
	if err != nil {
		return true
	} else {
		if conn != nil {
			_ = conn.Close()
			return false
		}
	}
	return true
}

func getDockerClient() (*client.Client, func(), error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, err
	}
	return cli, func() {
		_ = cli.Close()
	}, nil
}
