package apiserver

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/mirzakhany/dbctl/internal/container"
	"github.com/mirzakhany/dbctl/internal/logger"
)

const labelAPIServer = "apiserver"

// RunAPIServerContainer runs a container with the apiserver image
func RunAPIServerContainer(ctx context.Context, port, label string, timeout time.Duration) error {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		return err
	}

	req := container.CreateRequest{
		Image: "mirzakhani/dbctl:latest",
		Env: map[string]string{
			"DBCTL_INSIDE_DOCKER": "true",
		},
		Cmd:          []string{"/dbctl", "api-server"},
		ExposedPorts: []string{fmt.Sprintf("%s:1988/tcp", port)},
		Name:         fmt.Sprintf("dbctl_apiserver_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Labels:       map[string]string{container.LabelType: labelAPIServer},
	}

	if label != "" {
		req.Labels[container.LabelCustom] = label
	}

	_, err = container.Run(ctx, req)
	if err != nil {
		return err
	}

	// wait for the container port to be ready
	for {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), timeout)
		if err != nil {
			if err == context.DeadlineExceeded {
				return err
			}
		} else {
			_ = conn.Close()
			break
		}
	}

	logger.Info("Started apiserver on http://localhost:" + port)
	return nil
}
