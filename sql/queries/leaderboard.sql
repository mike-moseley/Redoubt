-- name: LeaderboardTop :many
SELECT * FROM leaderboard ORDER BY xp DESC, xp_updated_at ASC, id LIMIT @page_size;

-- name: LeaderboardAfter :many
SELECT * FROM leaderboard
WHERE xp <= @xp
	AND (
		xp < @xp
		OR xp_updated_at > @xp_updated_at
		OR (xp_updated_at = @xp_updated_at AND id > @id))
ORDER BY xp DESC, xp_updated_at ASC, id
LIMIT @page_size;
