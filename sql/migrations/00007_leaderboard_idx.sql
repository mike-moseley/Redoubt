-- +goose Up
CREATE INDEX characters_leaderboard_idx
	ON characters(xp DESC, xp_updated_at ASC, id ASC);

-- +goose Down
DROP INDEX characters_leaderboard_idx;
