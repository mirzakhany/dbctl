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
	"time"

	"github.com/docker/go-connections/nat"
)

const (
	LabelManagedBy = "managed_by"
	LabelDBctl     = "dbctl"
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
	res, err := callDockerAPI(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	return mapError(res)
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

	for _, pm := range exposedPortMap {
		for _, port := range pm {
			if !isPortFree(port.HostPort) {
				return "", fmt.Errorf("port: '%s' is already taken", port.HostPort)
			}
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("/%s/containers/create?name=%s", apiVersion, params.Name)
	res, err := callDockerAPI(ctx, http.MethodPost, path, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if err := mapError(res); err != nil {
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

	return re.ID, nil
}

func PullImage(ctx context.Context, image string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/images/create?fromImage=%s", apiVersion, image)
	res, err := callDockerAPI(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusOK {
		// read the body to make sure we wait for image to get pulled
		_, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
	}

	return mapError(res)
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
	res, err := callDockerAPI(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	if err := mapError(res); err != nil {
		return nil, err
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
	Status map[string][]string `json:"status"`
	Label  map[string]bool     `json:"label"`
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

// RemoveContainer removes a container by id
func RemoveContainer(ctx context.Context, id string) error {
	apiVersion, err := getAPIVersion(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/%s/containers/%s/kill?v=true&force=true&link=false", apiVersion, id)
	res, err := callDockerAPI(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}

	return mapError(res)
}

type errMessage struct {
	Message string `json:"message"`
}

func mapError(res *http.Response) error {
	switch res.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict:
		d, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		o := errMessage{}
		if err := json.Unmarshal(d, &o); err != nil {
			return err
		}

		return errors.New(o.Message)
	default:
		return nil
	}
}

func getAPIVersion(ctx context.Context) (string, error) {
	res, err := callDockerAPI(ctx, http.MethodGet, "/v1.20/version", nil)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	data := struct {
		APIVersion string `json:"apiVersion"`
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return "v" + data.APIVersion, nil
}

func callDockerAPI(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
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

	return client.Do(req.WithContext(ctx))
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

	socketAddr, err := getDockerSocketAddr()
	if err != nil {
		return nil, err
	}

	if strings.Contains(runtime.GOOS, "windows") {
		return &dockerAddr{
			addr:     socketAddr,
			protocol: "npipe",
			host:     "npipe:////" + socketAddr,
		}, nil
	} else if strings.Contains(runtime.GOOS, "unix") || strings.Contains(runtime.GOOS, "darwin") || strings.Contains(runtime.GOOS, "linux") {
		return &dockerAddr{
			addr:     socketAddr,
			protocol: "unix",
			host:     "unix://" + socketAddr,
		}, nil
	}
	return nil, errors.New("failed to get docker address")
}

func getDockerSocketAddr() (string, error) {
	if runtime.GOOS == "windows" {
		return "./pipe/docker_engine", nil
	}

	addr := "/var/run/docker.sock"

	// check if socker file is in /run/var directory otherwise check home directory
	if _, err := os.Stat(addr); err == nil {
		return addr, nil
	} else if os.IsNotExist(err) {
		// check if socker file is in home directory
		// get home directory path from os
		osHomeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		addr = fmt.Sprintf("%s/.docker/run/docker.sock", osHomeDir)
		if _, err := os.Stat(addr); err == nil {
			return addr, nil
		}

		return "", fmt.Errorf("docker socket file not found in /var/run/docker.sock or %s/.docker/run/docker.sock", osHomeDir)
	}

	return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
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
