package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPipeline(t *testing.T) {
	ctx := context.Background()

	b, err := setupBenthos(ctx)
	if err != nil {
		t.Fatalf("raised unexpected error: %q", err.Error())
	}
	defer b.Terminate(ctx)

	// fallback to default docker bridge gateway
	h := "172.17.0.1"
	if b.Hostname != "localhost" {
		h = b.Hostname
	}

	f, err := setupFluentBit(ctx, h, b.Port)
	if err != nil {
		t.Fatalf("raised unexpected error: %q", err.Error())
	}
	defer f.Terminate(ctx)

	time.Sleep(4 * time.Minute)
	t.Fail()
}

type benthos struct {
	tc.Container
	Hostname, Port string
}

func setupBenthos(ctx context.Context) (*benthos, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.New("setupBenthos, failed to get the working directory")
	}
	dir := path.Join(wd, "testdata", "benthos")

	fmt.Println(dir)

	req := tc.ContainerRequest{
		Image:        "jeffail/benthos",
		ExposedPorts: []string{"4195/tcp"},
		Mounts: tc.ContainerMounts{
			{
				Source:   tc.GenericBindMountSource{HostPath: dir},
				Target:   "/opt/benthos",
				ReadOnly: true,
			},
		},
		WaitingFor: wait.ForHTTP("/ready").WithPort("4195").WithStartupTimeout(10 * time.Second),
		Cmd:        []string{"-c", "/opt/benthos/benthos.yaml"},
	}

	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("setupBenthos, failed to start container: %w", err)
	}

	h, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("setupBenthos, failed to extract container IP: %w", err)
	}

	p, err := c.MappedPort(ctx, "4195/tcp")
	if err != nil {
		return nil, fmt.Errorf("setupBenthos, failed to extract container port: %w", err)
	}

	b := &benthos{
		Container: c,
		Hostname: h,
		Port: p.Port(),
	}

	return b, nil
}

func setupFluentBit(ctx context.Context, destHostname, destPort string) (tc.Container, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.New("setupFluentBit, failed to get the working directory")
	}
	dir := path.Join(wd, "testdata", "fluentbit")

	req := tc.ContainerRequest{
		Image:        "fluent/fluent-bit",
		ExposedPorts: []string{"2020/tcp"},
		Mounts: tc.ContainerMounts{
			{
				Source:   tc.GenericBindMountSource{HostPath: dir},
				Target:   "/opt/fluentbit",
				ReadOnly: true,
			},
		},
		Env: map[string]string{
			"HTTP_HOSTNAME": destHostname,
			"HTTP_PORT":     destPort,
		},
		WaitingFor: wait.ForHTTP("/api/v1/health").WithPort("2020").WithStartupTimeout(10 * time.Second),
		Cmd:        []string{"-c", "/opt/fluentbit/fluentbit.conf"},
	}

	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("setupFluentBit, failed to start container: %w", err)
	}

	return c, nil
}
