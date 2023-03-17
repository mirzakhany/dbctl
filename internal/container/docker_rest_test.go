package container

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"
)

func Test_StartContainer(t *testing.T) {
	id, err := CreateContainer(context.Background(), CreateRequest{
		Name: "test-rrr-00",
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "test",
		},
		Image:        "odidev/postgis:13-3.1-alpine",
		ExposedPorts: []string{"65436:5432/tcp"},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		Labels:       map[string]string{"foo": "bar"},
	})
	if err != nil {
		t.Fatalf("CreateContainer %s", err)
	}

	err = StartContainer(context.Background(), id)
	if err != nil {
		t.Fatalf("StartContainer %s", err)
	}
}

func Test_CreateContainer(t *testing.T) {
	var rnd, err = rand.Int(rand.Reader, big.NewInt(20))
	if err != nil {
		panic(err)
	}

	_, err = CreateContainer(context.Background(), CreateRequest{
		Name: fmt.Sprintf("dbctl_rs_%d_%d", time.Now().Unix(), rnd.Uint64()),
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "test",
		},
		Image:        "odidev/postgis:13-3.1-alpine",
		ExposedPorts: []string{"65436:5432/tcp"},
		Cmd:          []string{"postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		Labels:       map[string]string{"foo": "bar"},
	})

	if err != nil {
		t.Fatalf("CreateContainer %s", err)
	}
}

func Test_PullImage(t *testing.T) {
	err := PullImage(context.Background(), "odidev/postgis:13-3.1-alpine")
	if err != nil {
		t.Fatalf("PullImage %s", err)
	}
}

func Test_ListContainers(t *testing.T) {
	cn, err := List(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListContainers %s", err)
	}

	t.Log(cn)
}

func Test_RemoveContainer(t *testing.T) {
	err := RemoveContainer(context.Background(), "67dc394b62c1")
	if err != nil {
		t.Fatalf("removeContainer %s", err)
	}
}

func Test_KillContainer(t *testing.T) {
	err := KillContainer(context.Background(), "4ad7ff6cc5be")
	if err != nil {
		t.Fatalf("killContainer %s", err)
	}
}

func Test_getAPIVersion(t *testing.T) {
	v, err := getAPIVersion(context.Background())
	if err != nil {
		t.Fatalf("getAPIVersionfaild %s", err)
	}

	t.Log("docker api version", v)
}

func Test_isDockerRunning(t *testing.T) {
	o := isDockerRunning(context.Background())
	if o != true {
		t.Fatalf("docker is not running")
	}
}
