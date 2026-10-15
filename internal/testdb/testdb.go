package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func Start(ctx context.Context) (pool *pgxpool.Pool, cleanup func(), err error) {
	_, runDir, _, ok := runtime.Caller(0)
	if !ok {
		return nil, nil, errors.New("locating testdb source file")
	}

	fpDir := filepath.Dir(runDir)
	migrationsDir := filepath.Join(fpDir, "../../sql/migrations")

	ctr, err := postgres.Run(ctx, "postgres:18", postgres.WithDatabase("test_redoubt"),
		postgres.WithUsername("test"), postgres.WithPassword("test"),
		postgres.BasicWaitStrategies())
	defer func() {
		if err != nil {
			testcontainers.TerminateContainer(ctr)
		}
	}()
	if err != nil {
		return nil, nil, fmt.Errorf("starting container: %w", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("getting connection string: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("opening db: %w", err)
	}
	defer db.Close()

	err = goose.SetDialect("postgres")
	if err != nil {
		return nil, nil, fmt.Errorf("setting goose dialect: %w", err)
	}

	err = goose.Up(db, migrationsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("running migrations: %w", err)
	}

	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("opening pgx pool: %w", err)
	}
	return pool, func() {
		pool.Close()
		err := testcontainers.TerminateContainer(ctr)
		if err != nil {
			log.Printf("error closing test container: %v", err)
		}
	}, nil
}
