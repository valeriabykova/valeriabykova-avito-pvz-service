package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	dsn = "postgres://user:password@localhost:5432/postgres_db?sslmode=disable"
)

var (
	tables = []string{
		"users", "pvz", "reception", "product",
	}
)

var container *testcontainers.Container

func PreparePostgres(ctx context.Context, t *testing.T) *pgxpool.Pool {
	if container == nil {
		req := testcontainers.ContainerRequest{
			Image:        "postgres:15",
			ExposedPorts: []string{"5432:5432/tcp"},
			Hostname:     "localhost",
			Name:         "integrationPostgres",
			Env: map[string]string{
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "password",
				"POSTGRES_DB":       "postgres_db",
			},
			WaitingFor: wait.ForListeningPort(nat.Port("5432")),
		}
		pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		container = &pg
		if err != nil {
			t.Fatalf("failed to start container: %s", err.Error())
		}

		err = runMigrations(ctx, t)
		if err != nil {
			t.Fatalf("failed to run migrations: %s", err.Error())
		}
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Unable to create connection pool: %v\n", err)
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE %s RESTART IDENTITY", table))
		assert.NoError(t, err)
	}

	return pool
}

func runMigrations(ctx context.Context, t *testing.T) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to connect: %s", err.Error())
	}
	defer conn.Close()
	root, err := projectRoot()
	assert.NoError(t, err)
	return goose.Up(conn, fmt.Sprintf("%s/db/migrations", root))
}

func projectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}
	for ; ; current = filepath.Dir(current) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if entry.Name() == "go.mod" {
				return current, nil
			}
		}
		if current == "/" {
			return "", errors.New("module was not found")
		}
	}
}
