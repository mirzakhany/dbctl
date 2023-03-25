package container

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRestContainer(t *testing.T) {
	// -------------------------------------------------------------------------------
	// Pull the image
	// -------------------------------------------------------------------------------
	if err := PullImage(context.Background(), "busybox:latest"); err != nil {
		t.Fatalf("PullImage failed %s", err)
	}

	// -------------------------------------------------------------------------------
	// Create a container
	// -------------------------------------------------------------------------------
	name := fmt.Sprintf("test-%d", time.Now().Unix())
	id, err := CreateContainer(context.Background(), CreateRequest{
		Name:  name,
		Image: "busybox:latest",
		Cmd:   []string{"sh", "-c", "tail -f /dev/null"},
	})
	if err != nil {
		t.Fatalf("CreateContainer failed %s", err)
	}

	// -------------------------------------------------------------------------------
	// Start the container
	// -------------------------------------------------------------------------------
	if err := StartContainer(context.Background(), id); err != nil {
		t.Fatalf("StartContainer failed %s", err)
	}

	// TODO fix me, provide a functionality to wait for a docker
	time.Sleep(4 * time.Second)

	// -------------------------------------------------------------------------------
	// List running containers and look for the one we created
	// -------------------------------------------------------------------------------
	containers, err := List(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListContainers failed %s", err)
	}

	var found bool
	for _, c := range containers {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Failed to find our containers, running ones: %+v", containers)
	}

	// -------------------------------------------------------------------------------
	// Stop and remove the container
	// -------------------------------------------------------------------------------
	if err := RemoveContainer(context.Background(), id); err != nil {
		t.Fatalf("RemoveContainer failed %s", err)
	}
}
