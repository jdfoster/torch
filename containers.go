package main

import (
	"context"
	"fmt"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type container struct {
	tc.Container
	Hostname, Port, URI string
}

func newRedisContainer(ctx context.Context) (*container, error) {
	req := tc.ContainerRequest{
		Image:        "redis",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379").WithStartupTimeout(10 * time.Second),
	}

	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("setupRedis, failed to start container: %w", err)
	}

	h, err := c.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("setupRedis, failed to extract container IP: %w", err)
	}

	// fallback to default docker bridge gateway
	if h == "localhost" {
		h = "172.17.0.1"
	}

	p, err := c.MappedPort(ctx, "6379/tcp")
	if err != nil {
		return nil, fmt.Errorf("setupRedis, failed to extract container port: %w", err)
	}

	r := &container{
		Container: c,
		Hostname:  h,
		Port:      p.Port(),
		URI:       fmt.Sprintf("redis://%s:%s", h, p.Port()),
	}

	return r, nil
}

type benthosRequest struct {
	name, confPath string
	env            map[string]string
}

func newBenthosContainer(ctx context.Context, params benthosRequest) (*container, error) {
	req := tc.ContainerRequest{
		Name:         params.name,
		Image:        "jeffail/benthos",
		ExposedPorts: []string{"4195/tcp"},
		Mounts: tc.ContainerMounts{
			{
				Source:   tc.GenericBindMountSource{HostPath: params.confPath},
				Target:   "/opt/benthos",
				ReadOnly: true,
			},
		},
		Env:        params.env,
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
	// fallback to default docker bridge gateway
	if h == "localhost" {
		h = "172.17.0.1"
	}

	p, err := c.MappedPort(ctx, "4195/tcp")
	if err != nil {
		return nil, fmt.Errorf("setupBenthos, failed to extract container port: %w", err)
	}

	b := &container{
		Container: c,
		Hostname:  h,
		Port:      p.Port(),
		URI:       fmt.Sprintf("http://%s:%s", h, p.Port()),
	}

	return b, nil
}
