-- name: CreateCharacter :one
INSERT INTO characters (account_id, name)
VALUES (
	$1,
	$2
) RETURNING *;

-- name: GetCharacterByAccount :one
SELECT * FROM characters WHERE account_id = @account_id;
