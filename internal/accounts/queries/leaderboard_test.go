package queries_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

func createCharacterWithXP(t *testing.T, accountID uuid.UUID, name string, xp int64) queries.Character {
	t.Helper()
	ctx := t.Context()
	q := queries.New(pool)
	character, err := q.CreateCharacter(ctx, queries.CreateCharacterParams{
		AccountID: accountID,
		Name:      name,
	})
	if err != nil {
		t.Fatalf("Error creating character %q: %v", name, err)
	}

	_, err = q.ApplyStats(ctx, queries.ApplyStatsParams{
		XpDelta:     xp,
		GoldDelta:   0,
		DeathsDelta: 0,
		CharacterID: character.ID,
	})
	if err != nil {
		t.Fatalf("Error applying stats for %q: %v", name, err)
	}
	updatedCharacter, err := q.GetCharacterByAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("Error getting character %q by account: %v", name, err)
	}

	return updatedCharacter
}

func TestLeaderboardOrdering(t *testing.T) {
	const numChars = 5

	reset(t)
	ctx := t.Context()
	q := queries.New(pool)
	mike := createAccount(t, "mike")
	nick := createAccount(t, "nick")
	olivia := createAccount(t, "olivia")
	patrick := createAccount(t, "patrick")
	quillie := createAccount(t, "quillie")

	mikeChar := createCharacterWithXP(t, mike, "mike", 500)
	nickChar := createCharacterWithXP(t, nick, "nick", 400)
	oliviaChar := createCharacterWithXP(t, olivia, "olivia", 400)
	patrickChar := createCharacterWithXP(t, patrick, "patrick", 300)
	quillieChar := createCharacterWithXP(t, quillie, "quillie", 300)

	baseTime := time.Date(2020, time.April, 1, 1, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, "UPDATE characters SET xp_updated_at = $1 WHERE id = $2", baseTime, nickChar.ID)
	if err != nil {
		t.Fatalf("Error updating row in characters table: %v", err)
	}

	_, err = pool.Exec(ctx, "UPDATE characters SET xp_updated_at = $1 WHERE id = $2", baseTime.Add(5*time.Minute), oliviaChar.ID)
	if err != nil {
		t.Fatalf("Error updating row in characters table: %v", err)
	}

	_, err = pool.Exec(ctx, "UPDATE characters SET xp_updated_at = $1 WHERE id = $2", baseTime.Add(10*time.Minute), patrickChar.ID)
	if err != nil {
		t.Fatalf("Error updating row in characters table: %v", err)
	}
	_, err = pool.Exec(ctx, "UPDATE characters SET xp_updated_at = $1 WHERE id = $2", baseTime.Add(10*time.Minute), quillieChar.ID)
	if err != nil {
		t.Fatalf("Error updating row in characters table: %v", err)
	}

	lb, err := q.LeaderboardTop(ctx, numChars)
	if err != nil {
		t.Fatalf("Error retrieving leaderboard: %v", err)
	}

	if len(lb) != numChars {
		t.Fatalf("Incorrect leaderboard length: have %v, want %v", len(lb), numChars)
	}

	if lb[0].ID != mikeChar.ID {
		t.Fatalf("Leaderboard mismatch; have %v, want %v", lb[0].ID, mikeChar.ID)
	}

	if lb[1].ID != nickChar.ID {
		t.Fatalf("Leaderboard mismatch; have %v, want %v", lb[1].ID, nickChar.ID)
	}

	if lb[2].ID != oliviaChar.ID {
		t.Fatalf("Leaderboard mismatch; have %v, want %v", lb[2].ID, oliviaChar.ID)
	}

	if !lb[3].XpUpdatedAt.Equal(lb[4].XpUpdatedAt) {
		t.Fatalf("Expecting %v and %v to be equal", lb[3].XpUpdatedAt, lb[4].XpUpdatedAt)
	}
	// Postgres compares UUIDs by memcmp, so UUID that has a larger byte first
	// is greatest
	// We are sorting by Asc in the database, so smallest UUID is higher ranked
	if bytes.Compare(lb[3].ID[:], lb[4].ID[:]) >= 0 {
		t.Fatalf("Expecting %v to be smaller than %v", lb[3].ID, lb[4].ID)
	}
}

func TestLeaderboardPaging(t *testing.T) {
	reset(t)
	ctx := t.Context()
	q := queries.New(pool)

	acct1 := createAccount(t, "acct1")
	acct2 := createAccount(t, "acct2")
	acct3 := createAccount(t, "acct3")
	acct4 := createAccount(t, "acct4")
	acct5 := createAccount(t, "acct5")
	acct6 := createAccount(t, "acct6")
	acct7 := createAccount(t, "acct7")

	_ = createCharacterWithXP(t, acct1, "char1", 100)
	_ = createCharacterWithXP(t, acct2, "char2", 200)

	// Three identical characters that are larger than our
	// page size, boundary has to fall within a page.
	// Ensures correct behavior over the leaderboard index
	char3 := createCharacterWithXP(t, acct3, "char3", 300)
	char4 := createCharacterWithXP(t, acct4, "char4", 300)
	char5 := createCharacterWithXP(t, acct5, "char5", 300)

	_ = createCharacterWithXP(t, acct6, "char6", 600)
	_ = createCharacterWithXP(t, acct7, "char7", 700)

	baseTime := time.Date(2020, time.April, 1, 1, 0, 0, 0, time.UTC)
	tag, err := pool.Exec(ctx, "UPDATE characters SET xp_updated_at = $1 WHERE id = ANY($2)", baseTime, []uuid.UUID{char3.ID, char4.ID, char5.ID})
	if err != nil {
		t.Fatalf("Error updating row in characters table: %v", err)
	}
	if tag.RowsAffected() != 3 {
		t.Fatalf("Expected number of rows not updated: have %d, want %d", tag.RowsAffected(), 3)
	}

	expected, err := q.LeaderboardTop(ctx, 100)
	if err != nil {
		t.Fatalf("Error getting top leaderboard results: %v", err)
	}

	have := make([]queries.Leaderboard,0, 10)
	page, err := q.LeaderboardTop(ctx, 2)
	if err != nil {
		t.Fatalf("Error getting top leaderboard results: %v", err)
	}

	have = append(have, page...)

	done := false
	for range 10 {
		last := page[len(page)-1]
		page, err = q.LeaderboardAfter(ctx, queries.LeaderboardAfterParams{
			Xp:          last.Xp,
			XpUpdatedAt: last.XpUpdatedAt,
			ID:          last.ID,
			PageSize:    2,
		})
		if err != nil {
			t.Fatalf("Error getting top leaderboard results: %v", err)
		}
		if len(page) == 0 {
			done = true
			break
		}
		have = append(have, page...)
	}
	if !done {
		t.Fatalf("Did not finish querying leaderboard pages")
	}
	if len(have) != len(expected) {
		t.Fatalf("Leaderboard slice mismatch: have %d, want %d", len(have), len(expected))
	}
	for i, e := range expected {
		if e.ID != have[i].ID {
			t.Fatalf("Leaderboard entry mismatch:\n  have %+v\n want %+v", have[i], e)
		}
	}
}
