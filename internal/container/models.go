package container

import (
	"fmt"
	"os"
	"strings"
)

const (
	// LabelType is the label used to identify the type of database
	LabelType = "dbctl_type"
	// LabelCustom is the label used to identify a database
	LabelCustom = "dbctl_custom"

	// Connection details of the instance, recorded so that the api server can find
	// a running instance instead of assuming it listens on the default port. They
	// hold no more than `docker inspect` already shows.
	LabelPort = "dbctl_port"
	LabelUser = "dbctl_user"
	LabelPass = "dbctl_pass"
	LabelName = "dbctl_name"
)

// EnvListen overrides the address published ports are bound to.
const EnvListen = "DBCTL_LISTEN"

// LoopbackAddress is where published ports are bound unless asked otherwise.
const LoopbackAddress = "127.0.0.1"

// ListenAddress is the address published ports are bound to. Databases dbctl
// starts have well known credentials, so they stay on loopback unless the user
// explicitly asks for more.
func ListenAddress() string {
	if addr := strings.TrimSpace(os.Getenv(EnvListen)); addr != "" {
		return addr
	}
	return LoopbackAddress
}

// IsPublic reports whether the configured listen address exposes containers
// beyond this machine.
func IsPublic() bool {
	addr := ListenAddress()
	return addr != LoopbackAddress && addr != "localhost" && addr != "::1"
}

// PortSpec builds a docker port mapping that binds the host side to the
// configured address rather than to every interface. Docker's own rules sit in
// front of the host firewall, so a port published to 0.0.0.0 is reachable from
// the whole network even when the firewall says otherwise.
func PortSpec(hostPort, containerPort string) string {
	return fmt.Sprintf("%s:%s:%s", ListenAddress(), hostPort, containerPort)
}

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
