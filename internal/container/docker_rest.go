package container

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/docker/go-connections/nat"
)

const (
	LabelManagedBy = "managed_by"
	LabelDBctl     = "dbctl"
)

var (
	ErrNotFound   = errors.New("container not found")
	ErrNotRunning = errors.New("container is not running")
	ErrBadRequest = errors.New("bad request")
	ErrServer     = errors.New("server error")
)

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

func (c *Container) Terminate(ctx context.Context) error {
	return TerminateByID(ctx, c.ID)
}

func TerminateByID(ctx context.Context, id string) error {
	return RemoveContainer(ctx, id)
}

func StartContainer(ctx context.Context, id string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/containers/%s/start", apiVersion, id)
	res, err := callDockerApi(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	return mapError(res.StatusCode)
}

func CreateContainer(ctx context.Context, params CreateRequest) (string, error) {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return "", err
	}

	labels := map[string]string{LabelManagedBy: LabelDBctl}
	for k, v := range params.Labels {
		labels[k] = v
	}

	envs := make([]string, 0, len(params.Env))
	for k, v := range params.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(params.ExposedPorts)
	if err != nil {
		return "", err
	}

	req := DockerCreateConfig{
		Image:        params.Image,
		Cmd:          params.Cmd,
		Labels:       labels,
		Env:          envs,
		ExposedPorts: exposedPortSet,
		HostConfig:   HostConfig{PortBindings: exposedPortMap},
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("/%s/containers/create?name=%s", apiVersion, params.Name)
	res, err := callDockerApi(ctx, http.MethodPost, path, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	d, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read docker response failed: %w", err)
	}

	var re DockerCreateResponse
	err = json.NewDecoder(bytes.NewReader(d)).Decode(&re)
	if err != nil {
		return "", fmt.Errorf("read docker response failed: %w", err)
	}

	return re.ID, mapError(res.StatusCode)
}

func PullImage(ctx context.Context, image string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/images/create?fromImage=%s", apiVersion, image)
	res, err := callDockerApi(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	return mapError(res.StatusCode)
}

func List(ctx context.Context, labels map[string]string) ([]*Container, error) {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return nil, err
	}

	labels_ := map[string]string{LabelManagedBy: LabelDBctl}
	for k, v := range labels {
		labels_[k] = v
	}

	f, err := encodeFilters(labels_)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/%s/containers/json?limit=0&filters=%s", apiVersion, f)
	res, err := callDockerApi(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, mapError(res.StatusCode)
	}

	d, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read docker response failed: %w", err)
	}

	var containers []ListContainerResponse
	err = json.NewDecoder(bytes.NewReader(d)).Decode(&containers)
	if err != nil {
		return nil, fmt.Errorf("read docker response failed: %w", err)
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

type filters struct {
	Label map[string]bool `json:"label"`
}

func encodeFilters(args map[string]string) (string, error) {
	lb := make(map[string]bool)
	for k, v := range args {
		lb[fmt.Sprintf(`%s=%s`, k, v)] = true
	}

	d, err := json.Marshal(filters{Label: lb})
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func RemoveContainer(ctx context.Context, id string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/containers/%s/kill?v=true&force=true&link=false", apiVersion, id)
	res, err := callDockerApi(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	return mapError(res.StatusCode)
}

func KillContainer(ctx context.Context, id string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	res, err := callDockerApi(ctx, http.MethodPost, fmt.Sprintf("/%s/containers/%s/kill", apiVersion, id), nil)
	if err != nil {
		return err
	}

	return mapError(res.StatusCode)
}

func mapError(statusCode int) error {
	switch statusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrNotRunning
	default:
		return ErrServer
	}
}

func getAPIVersion(ctx context.Context) (string, error) {
	res, err := callDockerApi(ctx, http.MethodGet, "/v1.20/version", nil)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	data := struct {
		ApiVersion string `json:"apiVersion"`
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return "v" + data.ApiVersion, nil
}

func isDockerRunning(ctx context.Context) bool {
	res, err := callDockerApi(ctx, http.MethodGet, "/v1.20/version", nil)
	if err != nil || res.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func callDockerApi(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	addr, err := getDockerAddr()
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial(addr.protocol, addr.addr)
			},
		},
	}

	req, err := http.NewRequest(method, "http://localhost"+path, body)
	if err != nil {
		return nil, err
	}

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	req = req.WithContext(ctx)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type dockerAddr struct {
	addr     string
	protocol string
	host     string
}

func getDockerAddr() (*dockerAddr, error) {
	host := os.Getenv("DOCKER_HOST")
	if host != "" {
		hostURL, err := parseHostURL(host)
		if err != nil {
			return nil, err
		}
		return &dockerAddr{
			addr:     host,
			protocol: hostURL.Scheme,
			host:     hostURL.Host,
		}, nil
	}

	if strings.Contains(runtime.GOOS, "windows") {
		return &dockerAddr{
			addr:     "//./pipe/docker_engine",
			protocol: "npipe",
			host:     "npipe:////./pipe/docker_engine",
		}, nil
	} else if strings.Contains(runtime.GOOS, "unix") || strings.Contains(runtime.GOOS, "darwin") {
		return &dockerAddr{
			addr:     "/var/run/docker.sock",
			protocol: "unix",
			host:     "unix:///var/run/docker.sock",
		}, nil
	}
	return nil, errors.New("failed to get docker address")
}

// ParseHostURL parses a url string, validates the string is a host url, and
// returns the parsed URL
func parseHostURL(host string) (*url.URL, error) {
	protoAddrParts := strings.SplitN(host, "://", 2)
	if len(protoAddrParts) == 1 {
		return nil, fmt.Errorf("unable to parse docker host `%s`", host)
	}

	var basePath string
	proto, addr := protoAddrParts[0], protoAddrParts[1]
	if proto == "tcp" {
		parsed, err := url.Parse("tcp://" + addr)
		if err != nil {
			return nil, err
		}
		addr = parsed.Host
		basePath = parsed.Path
	}
	return &url.URL{
		Scheme: proto,
		Host:   addr,
		Path:   basePath,
	}, nil
}
