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

	cn, err := container.Run(ctx, req)
	if err != nil {
		return err
	}

	if err := waitForPort(ctx, port, timeout); err != nil {
		// the server is of no use if it never came up, take the container with it
		// instead of leaving it behind for the next run to trip over.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = container.TerminateByID(cleanupCtx, cn.ID)
		return err
	}

	logger.Info("Started apiserver on http://localhost:" + port)
	return nil
}

// waitForPort waits until the given port accepts connections, giving up once the
// timeout is reached rather than retrying forever.
func waitForPort(ctx context.Context, port string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	addr := net.JoinHostPort("127.0.0.1", port)
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("apiserver did not come up on %s within %s: %w", addr, timeout, err)
		case <-ticker.C:
		}
	}
}
