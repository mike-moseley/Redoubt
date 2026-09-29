-- name: CreateCharacter :one
INSERT INTO characters (account_id, name)
VALUES (
	$1,
	$2
) RETURNING *;

-- name: GetCharacterByAccount :one
SELECT * FROM characters WHERE account_id = @account_id;

-- name: ApplyStats :one
UPDATE characters
	SET xp = xp + @xp_delta,
	gold = gold + @gold_delta,
	deaths = deaths + @deaths_delta
WHERE id = @character_id
RETURNING level;
