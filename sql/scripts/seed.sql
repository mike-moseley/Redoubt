TRUNCATE accounts CASCADE;

INSERT INTO accounts (username, password_hash) SELECT 'user_' || n AS username, 'x' AS password_hash FROM generate_series(1, 500000) AS n;

INSERT INTO characters (account_id, name, xp, gold, deaths, xp_updated_at)
SELECT
	id,
	'hero_' || username AS name,
	(random() * 18000000)::bigint AS xp,
	(random() * 100000)::bigint AS gold,
	(random() * 100)::bigint AS deaths,
	now() - random() * interval '365 days' AS xp_updated_at
FROM accounts;
