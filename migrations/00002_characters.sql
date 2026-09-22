-- +goose Up

-- One character per account, and only stats that will be exposed for
-- leaderboard generation for v1
CREATE TABLE characters (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	account_id uuid NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
	name text NOT NULL UNIQUE,
	level int NOT NULL DEFAULT 1,
	xp bigint NOT NULL DEFAULT 0,
	gold bigint NOT NULL DEFAULT 0,
	deaths int NOT NULL DEFAULT 0,
	created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE characters;
