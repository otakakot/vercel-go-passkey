package testx

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/rueidis"
)

func SetupPostgres(
	t *testing.T,
	ddlpath string,
) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("failed to create new pool: %v", err)
	}

	if err := pool.Client.Ping(); err != nil {
		t.Fatalf("failed to ping: %v", err)
	}

	opt := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=test",
			"listen_addresses='*'",
		},
		Mounts: []string{
			ddlpath + ":/docker-entrypoint-initdb.d",
		},
	}

	hcOpt := func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	}

	hcOpts := []func(*docker.HostConfig){
		hcOpt,
	}

	resource, err := pool.RunWithOptions(&opt, hcOpts...)
	if err != nil {
		t.Fatalf("failed to run with options: %v", err)
	}

	port := resource.GetPort("5432/tcp")

	dsn := "postgres://postgres:postgres@localhost:" + port + "/test?sslmode=disable"

	if err := pool.Retry(func() error {
		conn, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		pool, err := pgxpool.NewWithConfig(context.Background(), conn)
		if err != nil {
			return fmt.Errorf("failed to create pool: %w", err)
		}

		if err := pool.Ping(context.Background()); err != nil {
			return fmt.Errorf("failed to ping: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	t.Cleanup(func() {
		if err := pool.Purge(resource); err != nil {
			t.Logf("failed to purge: %v", err)
		}
	})

	t.Setenv("POSTGRES_URL", dsn)
}

func SetupRedis(
	t *testing.T,
) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("failed to create new pool: %v", err)
	}

	resource, err := pool.Run("redis", "7-alpine", nil)
	if err != nil {
		t.Fatalf("failed to run redis: %v", err)
	}

	addr := net.JoinHostPort("localhost", resource.GetPort("6379/tcp"))

	if err := pool.Retry(func() error {
		cli, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}})
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ping := cli.B().Ping().Build()

		if err := cli.Do(context.Background(), ping).Error(); err != nil {
			return fmt.Errorf("failed to ping: %w", err)
		}

		return nil
	}); err != nil {
		t.Fatalf("Failed to ping Redis: %+v", err)
	}

	t.Cleanup(func() {
		if err := pool.Purge(resource); err != nil {
			t.Logf("failed to purge: %v", err)
		}
	})

	t.Setenv("KV_URL", addr)
}
