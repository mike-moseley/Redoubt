# Database

Postgres backs the accounts service (`cmd/accounts`, in progress): player
accounts, their character, and the stats that feed the leaderboard. The game server never
talks to Postgres directly. Schema changes are [goose](https://github.com/pressly/goose)
migrations in `migrations/`.

## Schema

```
accounts                         characters
--------                         ----------
id             uuid PK           id             uuid PK
username       text              account_id     uuid UNIQUE -> accounts.id (ON DELETE CASCADE)
password_hash  text              name           text
created_at     timestamptz       level, xp, gold, deaths
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
- **Non-negative stats.** `level >= 1`; `xp`, `gold` and `deaths >= 0`.

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
  including `characters.id`, so callers can order on the full key. Callers
  compute rank themselves, in the same query as their `ORDER BY ... LIMIT`,
  where the planner can see both together.
- **00007: add an index in leaderboard order.**
  ```sql
  CREATE INDEX characters_leaderboard_idx
  	ON characters (xp DESC, xp_updated_at ASC, id ASC);
  ```
  The column order and directions match the `ORDER BY` exactly. The
  directions are mixed, so an all-ascending index on the same columns would
  not work. All three columns are `NOT NULL`, so NULL ordering doesn't need to
  match.

The caller's query:

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

```sh
# schema
goose -dir migrations postgres "$DATABASE_URL" up

# 500k accounts + characters with random xp and xp_updated_at (TRUNCATEs first)
psql "$DATABASE_URL" -f scripts/seed.sql
ANALYZE;  # in psql, so the planner has fresh statistics

# before: roll back to the ranked view with no index
goose -dir migrations postgres "$DATABASE_URL" down-to 5
#   EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM leaderboard ORDER BY rank LIMIT 20;

# after
goose -dir migrations postgres "$DATABASE_URL" up
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
  Not wired up yet, because there's no `GET /leaderboard` handler.
- **`CREATE INDEX CONCURRENTLY`.** On a live table the index would be built
  concurrently, so writes aren't blocked while it builds. That can't run inside
  a transaction, so it needs `-- +goose NO TRANSACTION`. It isn't needed at
  this project's scale.
