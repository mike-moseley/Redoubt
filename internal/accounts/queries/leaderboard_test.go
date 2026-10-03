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
