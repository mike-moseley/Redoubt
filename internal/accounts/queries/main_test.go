package queries_test

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var pool *pgxpool.Pool

const migrationsDir = "../../../sql/migrations/"

const (
	usernameUnique      = "accounts_username_lower_idx"
	usernameLength      = "accounts_username_length"
	characterNameUnique = "characters_name_lower_idx"
	characterNameLength = "characters_name_length"
	goldNonNegative     = "characters_gold_nonneg"
	xpNonNegative       = "characters_xp_nonneg"
	deathsNonNegative   = "characters_deaths_nonneg"

	// Temporary
	// v1 only: multiple characters per account is a v2 feature
	oneCharacterPerAccount = "characters_account_id_key"
)

var nameLengthCases = []struct {
	name  string
	input string
	ok    bool
}{
	{name: "surrounding spaces", input: " mike ", ok: false},
	{name: "leading space", input: " mike", ok: false},
	{name: "trailing space", input: "mike ", ok: false},
	{name: "empty", input: "", ok: false},
	{name: "spaces", input: "  ", ok: false},
	{name: "long", input: strings.Repeat("m", 21), ok: false},
	{name: "lower bound", input: "m", ok: true},
	{name: "upper bound", input: strings.Repeat("m", 20), ok: true},
	{name: "correct", input: "mike", ok: true},
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func reset(t *testing.T) {
	t.Helper()
	_, err := pool.Exec(t.Context(), "TRUNCATE accounts CASCADE")
	if err != nil {
		t.Fatalf("Error resetting db for tests: %v", err)
	}
}

func createAccount(t *testing.T, username string) uuid.UUID {
	t.Helper()
	ctx := t.Context()
	q := queries.New(pool)
	row, err := q.CreateAccount(ctx, queries.CreateAccountParams{
		Username:     username,
		PasswordHash: "ver1table-smorg4sb0rd",
	})
	if err != nil {
		t.Fatalf("Error creating account: %v", err)
	}

	return row.ID
}

func expectPgError(t *testing.T, err error, code string, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if err == nil {
		t.Fatal("Command succeeded, should have failed")
	}
	if !errors.As(err, &pgErr) {
		t.Fatalf("Command failed with non-postgres error: %v", err)
	}
	if (pgErr.Code != code) || (pgErr.ConstraintName != constraint) {
		t.Fatalf("pgError mismatch: code = %q, want %q; constraint = %q, want %q",
			pgErr.Code, code, pgErr.ConstraintName, constraint)
	}
}

func run(m *testing.M) int {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:18", postgres.WithDatabase("test_redoubt"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())
	defer testcontainers.TerminateContainer(ctr)

	if err != nil {
		log.Printf("Error starting container: %v", err)
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
	defer db.Close()

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

	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Printf("Error opening pgx pool: %v", err)
		return 1
	}
	defer pool.Close()

	return m.Run()
}
