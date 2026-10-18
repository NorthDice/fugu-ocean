# Partitioning — Discussion Notes

Status: **open, not decided yet.** Captured here for later reference before implementing.

## Motivation

As users upload files, the `files` table grows unbounded. The idea is to use PostgreSQL
declarative partitioning to keep it faster and easier to manage as it grows.

## Context that shapes the decision

- File listing will use **keyset (cursor) pagination**, not `LIMIT`/`OFFSET` — pages are fetched
  via `WHERE created_at < :cursor ORDER BY created_at DESC LIMIT :n`, with older uploads showing
  first/later depending on direction. This means query patterns are already anchored on
  `created_at`.
- `files.id` is referenced by `shares.file_id` and `index_jobs.file_id` (see `docs/mvp.md` data
  model).

## Options considered

### 1. RANGE partitioning by `created_at` (e.g. monthly)

- Fits the keyset pagination pattern directly: `ORDER BY created_at` + a bound on `created_at`
  lets Postgres do a `MergeAppend` across partitions and stop once `LIMIT` is satisfied, without
  touching old partitions.
- New uploads land in the newest (small, hot) partition; older partitions become effectively
  read-only — cheaper to vacuum, and can later be moved to cold storage or archived by dropping
  the partition outright.
- Needs a mechanism to pre-create future partitions (`pg_partman`, or a small scheduled job/
  migration) plus a `DEFAULT` partition as a safety net for rows outside any defined range.
- Doesn't by itself prune on `folder_id`/`owner_id` — those need local indexes
  `(folder_id, created_at)` / `(owner_id, created_at)` on the partitioned table (Postgres
  propagates such indexes to all partitions automatically).

### 2. HASH partitioning by `owner_id`

- Matches "list my files" / "shared with me" access patterns (most queries are scoped by
  owner), but does **not** match the date-ordered keyset pagination requirement — first
  page still requires a merge across all partitions with no useful pruning.
- Rejected as primary strategy given the confirmed date-ordered pagination design, though could
  be revisited if access patterns change.

**Current leaning: RANGE by `created_at`, monthly**, given the pagination design. Not finalized.

## Known complication: partition key must be part of the primary key

PostgreSQL requires any `PRIMARY KEY`/`UNIQUE` constraint on a partitioned table to include the
partitioning column(s). So:

- `files` PK becomes composite: `PRIMARY KEY (id, created_at)` instead of just `id`.
- `shares.file_id` and `index_jobs.file_id` currently assume a plain FK to `files.id`. A
  partitioned `files` table can't offer a unique constraint on `id` alone, so a real FK needs
  `(file_id, created_at)` on both child tables — i.e. `created_at` would need to be denormalized
  onto `shares` and `index_jobs`.
- Alternative: drop the FK constraint on `files` entirely and enforce referential integrity in
  the service layer. Reasonable trade-off for a learning project, but should be a deliberate,
  documented choice rather than an accident.

## Sketch (illustrative, not final)

```sql
CREATE TABLE files (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    folder_id UUID,
    name TEXT NOT NULL,
    size BIGINT NOT NULL,
    content_type TEXT,
    storage_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX ON files (folder_id, created_at);
CREATE INDEX ON files (owner_id, created_at);

CREATE TABLE files_2026_09 PARTITION OF files
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE files_2026_10 PARTITION OF files
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE files_default PARTITION OF files DEFAULT;
```

## Open questions to resolve before implementing

1. Partition granularity — monthly assumed above; confirm it fits expected upload volume.
2. How new partitions get created ahead of time (extension like `pg_partman` vs. a hand-rolled
   scheduled job/migration).
3. FK strategy for `shares`/`index_jobs`: denormalize `created_at` onto them, or drop the FK and
   enforce integrity in the service layer.
4. Whether listing queries are ever scoped by `folder_id`/`owner_id` *without* a `created_at`
   bound — affects whether the local secondary indexes above are sufficient.
5. Whether partitioning is worth doing at MVP scale at all, vs. adding it later once real
   row/data volume justifies it (this is a learning project, so doing it early for the practice
   is also a valid reason — worth being explicit about which motivation applies).
