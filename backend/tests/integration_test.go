//go:build integration

// Package tests holds HTTP integration tests against a real PostgreSQL instance (Docker).
//
// Run from module root:
//
//	go test -tags=integration -v ./tests/...
//
// Requires Docker (testcontainers).
package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/taskflow/backend/internal/repository"
	"github.com/taskflow/backend/internal/router"
	"github.com/taskflow/backend/internal/service"
	"github.com/taskflow/backend/pkg/database"
)

const (
	testJWTSecret = "integration-test-jwt-secret-at-least-32-characters"
	testJWTHours  = 24
	testBcrypt    = 12
)

var (
	testDB     *sqlx.DB
	testEngine *gin.Engine
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container, dsn, err := startPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}
	defer func() {
		_ = container.Terminate(context.Background())
	}()

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer func() { _ = db.Close() }()

	migDir, err := migrationsDir()
	if err != nil {
		log.Fatalf("migrations dir: %v", err)
	}
	if err := runMigrations(dsn, migDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	testDB = db
	repos := repository.NewRepositories(db)
	svcs := service.NewServices(
		db,
		repos,
		testJWTSecret,
		testJWTHours*time.Hour,
		testBcrypt,
	)
	testEngine = router.SetupRouter(svcs, testJWTSecret)

	code := m.Run()
	os.Exit(code)
}

func startPostgresContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "taskflow",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(2 * time.Minute),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, "", err
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		return nil, "", err
	}
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/taskflow?sslmode=disable", host, port.Port())
	return c, dsn, nil
}

func migrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(wd, "..", "migrations"))
}

// runMigrations applies SQL files from dir using a dedicated *sql.DB.
// migrate's postgres driver Close() always closes the *sql.DB passed to WithInstance, so we must
// not use the application's pool — otherwise TestMain would leave testDB closed after m.Close().
// iofs + DirFS avoids file:// URL parsing issues on Windows.
func runMigrations(databaseURL, dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("migrations abs path: %w", err)
	}
	src, err := iofs.New(os.DirFS(abs), ".")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}

	migDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres for migrations: %w", err)
	}

	drv, err := postgres.WithInstance(migDB, &postgres.Config{})
	if err != nil {
		_ = migDB.Close()
		return fmt.Errorf("migrate postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", drv)
	if err != nil {
		_ = migDB.Close()
		return fmt.Errorf("migrate new: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
