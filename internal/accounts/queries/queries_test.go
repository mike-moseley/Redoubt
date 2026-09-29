package queries_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

var pool *pgxpool.Pool

const migrationsDir = "../../../sql/migrations/"

func TestMain(m *testing.M) {

	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:18", postgres.WithDatabase("test_redoubt"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())
	defer testcontainers.TerminateContainer(ctr)

	if err != nil {
		log.Printf("Error start container: %v", err)
		return 1
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("Error getting connection string: %v", err)
		return 1
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("Error opening db: %v", err)
		return 1
	}

	err = goose.SetDialect("postgres")
	if err != nil {
		log.Printf("Error setting goose dialect: %v", err)
		return 1
	}

	err = goose.Up(db, migrationsDir)
	if err != nil {
		log.Printf("Error bringing up migrations: %v", err)
		return 1
	}
	db.Close()

	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Printf("Error opening pgx pool: %v", err)
		return 1
	}
	defer pool.Close()

	return m.Run()
}

func TestCaseInsensitive(t *testing.T) {
	ctx := t.Context()
	q := queries.New(pool)
	row, err := q.CreateAccount(ctx, queries.CreateAccountParams{
		Username: "mike",
		PasswordHash: "ver1table-smorg4sb0rd",
	})
	if err != nil {
		t.Fatalf("Error creating account: %v", err)
	}
	account, err := q.GetAccountByUsername(ctx, "Mike")
	if err != nil {
		t.Fatalf("Error getting account: %v", err)
	}
	if row.ID != account.ID {
		t.Fatalf("ID = %v, want %v", account.ID, row.ID)
	}
}
