-- +goose Up
DROP VIEW leaderboard;

ALTER TABLE characters DROP COLUMN level;

ALTER TABLE characters ADD COLUMN level
	int NOT NULL GENERATED ALWAYS AS
	(LEAST(100, floor(7 * log(2, (xp::numeric/1000 + 1))) + 1)::int) STORED;

CREATE VIEW leaderboard AS
	SELECT
		accounts.username, characters.name, characters.level, characters.xp,
		characters.xp_updated_at, characters.id
	FROM characters JOIN accounts ON characters.account_id = accounts.id;

-- +goose Down
DROP VIEW leaderboard;

ALTER TABLE characters
	ALTER COLUMN level DROP EXPRESSION,
	ALTER COLUMN level SET DEFAULT 1,
	ADD CONSTRAINT characters_level_min CHECK(level>=1);

CREATE VIEW leaderboard AS
	SELECT
		accounts.username, characters.name, characters.level, characters.xp,
		characters.xp_updated_at, characters.id
	FROM characters JOIN accounts ON characters.account_id = accounts.id;
