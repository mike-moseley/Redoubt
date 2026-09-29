-- +goose Up
CREATE TABLE accounts (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	username text NOT NULL,
	password_hash text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

-- Enforces uniqueness of account username regardless of case
CREATE UNIQUE INDEX accounts_username_lower_idx ON accounts(lower(username));

-- +goose Down
DROP TABLE accounts;
