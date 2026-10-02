package queries_test

import (
	"fmt"
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

func TestAccountDeleteCascades(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	id := createAccount(t, "mike")
	ch, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}
	_, err = pool.Exec(ctx, "DELETE FROM accounts WHERE id = $1", id)
	if err != nil {
		t.Fatalf("Error deleting account: %v", err)
	}
	var n int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM characters WHERE id = $1", ch.ID).Scan(&n)
	if err != nil {
		t.Fatalf("Error querying row in characters: %v", err)
	}
	if n != 0 {
		t.Fatalf("characters rows = %d, want 0", n)
	}
}

func TestLevelThresholds(t *testing.T) {
	levelCases := []struct {
		xp    int64
		level int32
	}{
		{xp: 0, level: 1},
		{xp: 1438, level: 9},
		{xp: 1439, level: 10},
		{xp: 127000, level: 50},
		{xp: 18088402, level: 99},
		{xp: 18088403, level: 100},
		{xp: 1 << 40, level: 100},
	}

	for _, tc := range levelCases {
		t.Run(fmt.Sprintf("xp %d", tc.xp), func(t *testing.T) {
			reset(t)
			ctx := t.Context()
			q := queries.New(pool)
			id := createAccount(t, "mike")

			ch, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: "hero"})
			if err != nil {
				t.Fatalf("Error creating character: %v", err)
			}
			if ch.Level != 1 {
				t.Fatalf("Character level incorrect: have %d, want 1", ch.Level)
			}

			level, err := q.ApplyStats(ctx, queries.ApplyStatsParams{
				XpDelta:     tc.xp,
				GoldDelta:   0,
				DeathsDelta: 0,
				CharacterID: ch.ID,
			})
			if err != nil {
				t.Fatalf("Error applying stats: %v", err)
			}
			if level != tc.level {
				t.Fatalf("Level mismatch; have %d, want %d", level, tc.level)
			}
		})
	}
}

func TestApplyStatsAddsDeltas(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	id := createAccount(t, "mike")

	const (
		wantXp     = 101
		wantGold   = 0
		wantDeaths = 4
	)

	ch, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     1,
		GoldDelta:   100,
		DeathsDelta: 3,
		CharacterID: ch.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     100,
		GoldDelta:   -100,
		DeathsDelta: 1,
		CharacterID: ch.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats: %v", err)
	}

	ch1, err := q.GetCharacterByAccount(ctx, ch.AccountID)
	if err != nil {
		t.Fatalf("Error getting character by account: %v", err)
	}

	if (ch1.Xp != wantXp) || (ch1.Gold != wantGold) || (ch1.Deaths != wantDeaths) {
		t.Fatalf("Unexpected stat totals:\n  XP: have %d want %d\n  Gold: have %d want %d\n  Deaths: have %d want %d", ch1.Xp, wantXp, ch1.Gold, wantGold, ch1.Deaths, wantDeaths)
	}
}

func TestApplyStatsGoldOverdraft(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	id := createAccount(t, "mike")

	ch, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     100,
		GoldDelta:   10,
		DeathsDelta: 1,
		CharacterID: ch.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats: %v", err)
	}

	ch1, err := q.GetCharacterByAccount(ctx, ch.AccountID)
	if err != nil {
		t.Fatalf("Error getting character by account: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     100,
		GoldDelta:   -11,
		DeathsDelta: 1,
		CharacterID: ch.ID,
	})
	expectPgError(t, err, pgerrcode.CheckViolation, goldNonNegative)

	ch2, err := q.GetCharacterByAccount(ctx, ch.AccountID)
	if err != nil {
		t.Fatalf("Error getting character by account: %v", err)
	}

	if ch1 != ch2 {
		t.Fatalf("row changed: \n got %+v\n want %+v", ch2, ch1)
	}
}

func TestApplyStatsXpUpdatedAt(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	id := createAccount(t, "mike")

	ch, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{AccountID: id, Name: "hero"})
	if err != nil {
		t.Fatalf("Error creating character: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     0,
		GoldDelta:   100,
		DeathsDelta: 3,
		CharacterID: ch.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats: %v", err)
	}

	ch1, err := q.GetCharacterByAccount(ctx, ch.AccountID)
	if err != nil {
		t.Fatalf("Error getting character by account: %v", err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     100,
		GoldDelta:   0,
		DeathsDelta: 0,
		CharacterID: ch.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats: %v", err)
	}

	ch2, err := q.GetCharacterByAccount(ctx, ch.AccountID)
	if err != nil {
		t.Fatalf("Error getting character by account: %v", err)
	}

	if !ch1.XpUpdatedAt.Equal(ch.XpUpdatedAt) {
		t.Fatalf("XpUpdatedAt mismatch: have %v, want %v", ch1.XpUpdatedAt, ch.XpUpdatedAt)
	}

	if !ch2.XpUpdatedAt.After(ch1.XpUpdatedAt) {
		t.Fatalf("XpUpdatedAt = %v, want after %v", ch2.XpUpdatedAt, ch1.XpUpdatedAt)
	}
}
