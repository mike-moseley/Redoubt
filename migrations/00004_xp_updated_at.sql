-- +goose Up
ALTER TABLE characters
	ADD COLUMN xp_updated_at timestamptz NOT NULL DEFAULT now();

UPDATE characters SET xp_updated_at = created_at;

-- +goose StatementBegin
CREATE FUNCTION set_xp_updated_at() RETURNS trigger AS $$
BEGIN
	NEW.xp_updated_at := now();
	return NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER characters_xp_updated
	BEFORE UPDATE OF xp ON characters
	FOR EACH ROW
	WHEN (OLD.xp IS DISTINCT FROM NEW.xp)
	EXECUTE FUNCTION set_xp_updated_at();

-- +goose Down
DROP TRIGGER characters_xp_updated ON characters;

DROP FUNCTION set_xp_updated_at();

ALTER TABLE characters DROP COLUMN xp_updated_at;
