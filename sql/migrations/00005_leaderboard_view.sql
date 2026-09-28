-- +goose Up
CREATE VIEW leaderboard AS 
	SELECT
		ROW_NUMBER() OVER (ORDER BY characters.xp DESC, characters.xp_updated_at ASC, characters.id ) AS rank,
		accounts.username, characters.name, characters.level, characters.xp, characters.xp_updated_at
	FROM characters JOIN accounts ON characters.account_id = accounts.id;

-- +goose Down
DROP VIEW leaderboard;
