package queries_test

import (
	"strings"
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

func TestCreateAccountUsernameLength(t *testing.T) {
	cases := []struct {
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			ctx := t.Context()
			q := queries.New(pool)

			_, err := q.CreateAccount(ctx, queries.CreateAccountParams{
				Username:     tc.input,
				PasswordHash: "asdfzxcv",
			})
			if !tc.ok {
				expectPgError(t, err, pgerrcode.CheckViolation, usernameLength)
				return
			}
			if err != nil {
				t.Fatalf("Error creating account: %v", err)
			}
		})
	}
}

