-- name: CreateAccount :one
INSERT INTO accounts (username, password_hash)
VALUES(
	$1,
	$2
) RETURNING id, username, created_at;

-- name: GetAccountByUsername :one
SELECT * FROM accounts WHERE lower(username) = lower(@username);
