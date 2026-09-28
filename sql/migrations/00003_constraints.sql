-- +goose Up
ALTER TABLE characters DROP CONSTRAINT characters_name_key;

CREATE UNIQUE INDEX characters_name_lower_idx ON characters(lower(name));

ALTER TABLE characters
	ADD CONSTRAINT characters_name_length CHECK(name = btrim(name) AND length(name) BETWEEN 1 AND 20),
	ADD CONSTRAINT characters_level_min CHECK(level>=1),
	ADD CONSTRAINT characters_xp_nonneg CHECK(xp>=0),
	ADD CONSTRAINT characters_gold_nonneg CHECK(gold>=0),
	ADD CONSTRAINT characters_deaths_nonneg CHECK(deaths>=0);

ALTER TABLE accounts
	ADD CONSTRAINT accounts_username_length CHECK(username = btrim(username) AND length(username) BETWEEN 1 AND 20);

-- +goose Down
ALTER TABLE accounts
	DROP CONSTRAINT accounts_username_length;

ALTER TABLE characters
	DROP CONSTRAINT characters_name_length,
	DROP CONSTRAINT characters_level_min,
	DROP CONSTRAINT characters_xp_nonneg,
	DROP CONSTRAINT characters_gold_nonneg,
	DROP CONSTRAINT characters_deaths_nonneg;

DROP INDEX characters_name_lower_idx;

ALTER TABLE characters ADD CONSTRAINT characters_name_key UNIQUE (name);
