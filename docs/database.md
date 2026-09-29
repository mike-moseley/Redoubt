# Database

Postgres backs the accounts service (`cmd/accounts`, in progress): player
accounts, their character, and the stats that feed the leaderboard. The game server never
talks to Postgres directly. Schema changes are [goose](https://github.com/pressly/goose)
migrations in `sql/migrations/`. Queries live in `sql/queries/` and are compiled
to typed Go by [sqlc](https://sqlc.dev) into `internal/accounts/queries`.

## Schema

```
accounts                         characters
--------                         ----------
id             uuid PK           id             uuid PK
username       text              account_id     uuid UNIQUE -> accounts.id (ON DELETE CASCADE)
password_hash  text              name           text
created_at     timestamptz       level          int  (generated from xp)
                                 xp             bigint
                                 gold           bigint
                                 deaths         int
                                 xp_updated_at  timestamptz
                                 created_at     timestamptz
```

- **One character per account (v1).** Enforced by `UNIQUE (account_id)` rather
  than by application code.
- **UUID keys** (`gen_random_uuid()`).
- **Deleting an account deletes its character** (`ON DELETE CASCADE`).

## Constraints (00001, 00003)

Integrity rules live in the database so they hold for every writer: the
accounts service, the stats endpoint, and anyone with a `psql` prompt.

- **Case-insensitive unique names.** Unique indexes on `lower(username)` and
  `lower(name)` stop `Bob` and `bob` from both existing. 00003 replaces the
  plain `UNIQUE (name)` from 00002, which only compared exact case.
- **Trimmed, bounded lengths.** `CHECK (name = btrim(name) AND length(name) BETWEEN 1 AND 20)`,
  and the same rule for `username`. This rejects blank names and names that
  differ only by whitespace.
- **Non-negative stats.** `xp`, `gold` and `deaths >= 0`. (00003 also had
  `level >= 1`; 00008 dropped it when `level` became a generated column whose
  formula can't go below 1.)

## Deterministic ranking (00004)

Leaderboard order is `xp DESC, xp_updated_at ASC, id ASC`:

1. Highest XP first.
2. On an XP tie, whoever reached it **first** ranks higher.
3. `id` breaks any remaining tie, so the order is total: the same data always
   produces the same ranking.

`xp_updated_at` is maintained by a trigger rather than by callers:

```sql
CREATE TRIGGER characters_xp_updated
	BEFORE UPDATE OF xp ON characters
	FOR EACH ROW
	WHEN (OLD.xp IS DISTINCT FROM NEW.xp)
	EXECUTE FUNCTION set_xp_updated_at();
```

The `WHEN` guard matters: an update that writes the same XP value back (for
example, a stats push that sets every column but only `gold` changed) must
not bump the timestamp,
or the player would lose their tiebreak position for no reason.

## Level from XP (00008)

`level` is a stored generated column, so the database computes it and nothing
else can write it:

```sql
level int NOT NULL GENERATED ALWAYS AS
	(LEAST(100, floor(7 * log(2, (xp::numeric/1000 + 1))) + 1)::int) STORED
```

Level is derived from XP. As a separate writable column it could drift out of
sync with XP. Now Postgres rejects any write to it, and Go code only reads it.

The curve is exponential, like RuneScape's: XP to reach level `L` is
`1000 · (2^((L−1)/7) − 1)`, so the cost doubles every 7 levels. Around 90
it becomes a soft cap without a separate rule, and 100 is a hard cap
(`LEAST`):

| Level | Total XP   |
|------:|-----------:|
| 10    | 1,439      |
| 50    | 127,000    |
| 90    | 6,719,188  |
| 99    | 16,383,000 |
| 100   | 18,088,403 |

Level 92 is half of 99, as in RuneScape, and levels 90 to 100 cost about 63%
of all XP.

Details:
- **`STORED` is spelled out.** Since PostgreSQL 18, a generated column
  defaults to `VIRTUAL`, which is computed on every read.
- **`xp::numeric`**: `xp` is `bigint`, so without the cast `xp/1000`
  would silently truncate. `log(2, x)` is used because Postgres has no `log2()`.
- **The down migration keeps the values.**
  `ALTER COLUMN level DROP EXPRESSION` turns it back into a plain column and
  keeps each row's computed level. Dropping and re-adding it would reset
  everyone to 1.
- **Tuning.** Changing the curve is a new migration
  (`ALTER COLUMN level SET EXPRESSION`); XP is the source of truth, so nothing
  is lost. A hand-tuned per-level table would need a trigger instead, because
  a generated column can't read other tables.

## Stats updates

The game server reports progress through one query:

```sql
-- name: ApplyStats :one
UPDATE characters
	SET xp = xp + @xp_delta,
	gold = gold + @gold_delta,
	deaths = deaths + @deaths_delta
WHERE id = @character_id
RETURNING level;
```

- **Deltas, not totals.** The game server never reads a character's totals.
  It doesn't call the accounts service when a player connects, so the
  realtime path doesn't depend on it. It can only report changes.
  `RETURNING level` shows the caller whether the push caused a level-up.
- **One push per game event** (a kill sends XP, a purchase sends gold), with no
  batching, so a failed push loses only that one event.
- **An overdraft is rejected, not clamped.** If a delta would take gold below
  zero, `characters_gold_nonneg` fails the whole update. Clamping with
  `GREATEST(0, ...)` would let a 1-gold player buy a 20-gold item and keep
  the other 19 as free gold.
- **Known v1 gaps.** A retried push is applied twice (the fix is deduplicating
  by push ID). The game server can't check a balance before a purchase, so the
  constraint is the only guard (the fix is putting starting stats into the
  login JWT's claims).

## Testing

Integration tests in `internal/accounts/queries/queries_test.go` run against a
real `postgres:18` container started by
[testcontainers-go](https://golang.testcontainers.org/). `TestMain` starts one
container for the package, applies the goose migrations, and shares a
`pgxpool` with the tests. The behaviour under test is Postgres's own:
constraints, the trigger, and the generated column. A mock would test none of
it.

```sh
go test ./internal/accounts/queries -v   # needs a running Docker daemon
```

## Leaderboard performance (00005 → 00007)

### The problem

The first leaderboard view (00005) computed rank inside the view:

```sql
CREATE VIEW leaderboard AS
	SELECT
		ROW_NUMBER() OVER (ORDER BY characters.xp DESC, characters.xp_updated_at ASC, characters.id) AS rank,
		...
	FROM characters JOIN accounts ON characters.account_id = accounts.id;
```

A top-20 query (`SELECT * FROM leaderboard ORDER BY rank LIMIT 20`) against
500k seeded characters took **241 ms** (warm). The plan shows why:

```
Limit  (actual time=228.297..233.202 rows=20)
  ->  Sort  (Sort Key: leaderboard.rank, top-N heapsort)
        ->  Subquery Scan on leaderboard  (rows=500000)
              ->  WindowAgg  (rows=500000)
                    ->  Gather Merge  (rows=500000)
                          ->  Sort  (Sort Method: external merge  Disk: 12440kB)   -- x3 workers
                                ->  Parallel Hash Join
                                      ->  Parallel Seq Scan on characters
                                      ->  Parallel Hash -> Parallel Seq Scan on accounts
```

Every row's rank depends on every other row, so Postgres has to join and sort
all 500k characters and number them before the outer `LIMIT` gets to run. The
`LIMIT` can't be pushed down past the window function, and an index can't help
because the planner has no way to stop early. The sort also exceeded
`work_mem` (4 MB default) and spilled roughly 37 MB to disk across three
workers.

Raising `work_mem` would keep the sort in memory, but it would still sort
every row on every request. The fix is to stop sorting altogether.

### The fix

- **00006: remove `rank` from the view.** The view now returns plain rows,
  including `characters.id`, so callers can order on the full key. Rank is
  just a row's position in the ordered result, so the application computes it
  (`offset + i + 1`). That works the same on every keyset page, where a
  `ROW_NUMBER()` in the query would restart at 1.
- **00007: add an index in leaderboard order.**
  ```sql
  CREATE INDEX characters_leaderboard_idx
  	ON characters (xp DESC, xp_updated_at ASC, id ASC);
  ```
  The column order and directions match the `ORDER BY` exactly. The
  directions are mixed, so an all-ascending index on the same columns would
  not work. All three columns are `NOT NULL`, so NULL ordering doesn't need to
  match.

The benchmark query still computes rank in SQL, to show that even a
SQL-side rank is cheap once it sits in the same query as `ORDER BY ... LIMIT`:

```sql
SELECT ROW_NUMBER() OVER (ORDER BY xp DESC, xp_updated_at, id) AS rank, *
FROM leaderboard
ORDER BY xp DESC, xp_updated_at, id
LIMIT 20;
```

```
Limit  (actual time=0.016..0.055 rows=20)
  ->  WindowAgg  (rows=20)
        ->  Nested Loop  (rows=20)
              ->  Index Scan using characters_leaderboard_idx on characters  (rows=20)
              ->  Index Scan using accounts_pkey on accounts  (rows=1, loops=20)
```

The planner reads the first 20 entries from the index (already in order),
looks up 20 accounts by primary key, numbers them, and stops. The planner's
estimate says `rows=500000` for the index scan; the actual `rows=20` shows the
`Limit` stopped it early.

### Results

500k characters, PostgreSQL 18.6, default settings (`shared_buffers` 128 MB,
`work_mem` 4 MB), laptop. Each query ran three times before the measured run.

|                         | Before (00005)                     | After (00007)           |
|-------------------------|------------------------------------|-------------------------|
| Execution time          | 241 ms                             | 0.084 ms                |
| Rows processed          | 500,000 joined, sorted, ranked     | 20                      |
| Table pages read        | 11,904                             | 103                     |
| Temp (disk) I/O         | ~26k pages read + written          | none                    |
| Plan shape              | Seq scans → hash join → disk sort → WindowAgg → Sort → Limit | Index scan → nested loop → WindowAgg → Limit |

Even warm, the "before" plan reports most of its pages as `read` rather than
`hit`. A sequential scan of a table larger than a quarter of `shared_buffers`
(`characters` is 60 MB, `accounts` 33 MB) goes through a small ring buffer, so
Postgres never keeps those tables in its own cache. Those reads come from the
OS page cache. Repeated full scans stay expensive no matter how often they
run.

The first measurement, taken cold, was 323 ms.

## Reproducing

goose reads its driver, connection string and migration directory from `.env`
(copy `.env.example`).

```sh
# schema
goose up

# 500k accounts + characters with random xp and xp_updated_at (TRUNCATEs first)
psql "$GOOSE_DBSTRING" -f sql/scripts/seed.sql
ANALYZE;  # in psql, so the planner has fresh statistics

# before: roll back to the ranked view with no index
goose down-to 5
#   EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM leaderboard ORDER BY rank LIMIT 20;

# after
goose up
#   EXPLAIN (ANALYZE, BUFFERS)
#   SELECT ROW_NUMBER() OVER (ORDER BY xp DESC, xp_updated_at, id) AS rank, *
#   FROM leaderboard ORDER BY xp DESC, xp_updated_at, id LIMIT 20;
```

## Not done yet

- **Keyset pagination.** Later pages should continue from the last row's
  `(xp, xp_updated_at, id)` rather than use `OFFSET`, which reads and discards
  every earlier row. Because the sort directions are mixed, a row comparison
  like `(xp, xp_updated_at, id) < (...)` doesn't match the order. The
  condition has to be spelled out, and it needs a leading `xp <= $1` so the
  index scan can *start* at the cursor instead of filtering from the top:
  ```sql
  WHERE xp <= $1
    AND (xp < $1 OR xp_updated_at > $2 OR (xp_updated_at = $2 AND id > $3))
  ```
  Measured on the page starting at row 19,001: `OFFSET` touched ~96k buffers.
  The `OR` chain without the `xp <= $1` bound used the index, but it still
  read and discarded 19,001 rows (~19k buffers). With the bound, the plan
  shows `Index Cond: (xp <= ...)`, removes 2 rows, and touches 108 buffers.
  This is the `LeaderboardAfter` sqlc query (`LeaderboardTop` serves the
  first page). Nothing calls it yet, because there's no `GET /leaderboard`
  handler.
- **More integration tests.** Only the case-insensitive username lookup is
  covered so far. Still to come: duplicate names, length checks, level
  thresholds, the `xp_updated_at` trigger, the ApplyStats overdraft, and
  keyset paging across ties.
- **`CREATE INDEX CONCURRENTLY`.** On a live table the index would be built
  concurrently, so writes aren't blocked while it builds. That can't run inside
  a transaction, so it needs `-- +goose NO TRANSACTION`. It isn't needed at
  this project's scale.

