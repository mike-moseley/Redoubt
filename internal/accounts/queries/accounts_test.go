package queries_test

import (
	"testing"

	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

func TestGetAccountByUsernameCaseInsensitive(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	row, err := q.CreateAccount(ctx, queries.CreateAccountParams{
		Username:     "mike",
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

func TestCreateAccountDuplicateUsernameDifferentCase(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)

	_, err := q.CreateAccount(ctx, queries.CreateAccountParams{
		Username:     "mike",
		PasswordHash: "asdfzxcv",
	})
	if err != nil {
		t.Fatalf("Error creating account: %v", err)
	}

	_, err = q.CreateAccount(ctx, queries.CreateAccountParams{
		Username:     "Mike",
		PasswordHash: "asdfqwer",
	})

	expectPgError(t, err, pgerrcode.UniqueViolation, usernameUnique)
}
