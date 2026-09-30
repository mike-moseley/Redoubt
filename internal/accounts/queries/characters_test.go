package queries_test

import (
	"testing"

	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

func TestOneCharacterPerAccount(t *testing.T) {
	reset(t)
	q := queries.New(pool)
	id := createAccount(t, "mike")
	_, err := q.CreateCharacter(t.Context(), queries.CreateCharacterParams{AccountID: id, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}
	_, err = q.CreateCharacter(t.Context(), queries.CreateCharacterParams{AccountID: id, Name: "zero"})
	expectPgError(t, err, pgerrcode.UniqueViolation, oneCharacterPerAccount)
}

func TestCharacterNameUnique(t *testing.T) {
	reset(t)
	q := queries.New(pool)
	mike := createAccount(t, "mike")
	dave := createAccount(t, "dave")
	_, err := q.CreateCharacter(t.Context(), queries.CreateCharacterParams{AccountID: mike, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}
	_, err = q.CreateCharacter(t.Context(), queries.CreateCharacterParams{AccountID: dave, Name: "Hero"})
	expectPgError(t, err, pgerrcode.UniqueViolation, characterNameUnique)
}

func TestCreateCharacterNameLength(t *testing.T) {
	for _, tc := range nameLengthCases {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			ctx := t.Context()
			q := queries.New(pool)
			id := createAccount(t, "mike")

			_, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: tc.input})
			if !tc.ok {
				expectPgError(t, err, pgerrcode.CheckViolation, characterNameLength)
				return
			}
			if err != nil {
				t.Fatalf("Error creating character: %v", err)
			}
		})
	}
}
