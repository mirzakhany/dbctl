package container

import "github.com/docker/go-connections/nat"

type Container struct {
	ID     string
	Name   string
	Labels map[string]string
}

type CreateRequest struct {
	Name         string
	Image        string
	ExposedPorts []string // allow specifying protocol info
	Cmd          []string
	Env          map[string]string
	Labels       map[string]string
}

type DockerCreateConfig struct {
	Image        string            `json:"Image"`
	Cmd          []string          `json:"Cmd"`
	Labels       map[string]string `json:"Labels"`
	Env          []string          `json:"Env"`
	ExposedPorts nat.PortSet       `json:"ExposedPorts"`
	HostConfig   HostConfig        `json:"HostConfig"`
}

type DockerCreateResponse struct {
	ID string `json:"Id"`
}

type HostConfig struct {
	PortBindings nat.PortMap
}

type ListContainerResponse struct {
	ID     string `json:"Id"`
	Names  []string
	Labels map[string]string
}
