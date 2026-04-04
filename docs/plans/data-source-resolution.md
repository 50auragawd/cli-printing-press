# Data Source Resolution: Live-First with Honest Local Fallback

## The Problem in Plain Terms

Every printed CLI talks to a remote API. When you run `orders list`, it calls the API and shows you results. This is straightforward and works the way anyone would expect.

Some printed CLIs also get a `search` command — full-text search across your data. But search doesn't call the API. It queries a local database on your machine. That database starts empty. You have to manually run a separate `sync` command first to pull data down from the API into the local database. If you forget, search errors. If you synced last week, search silently shows you week-old data with no indication it's stale. If something was deleted on the server since your last sync, search still returns it.

The fundamental issue: **the CLI has two completely separate data paths — live API and local database — with no coordination between them.** Regular commands always go live. Search always goes local. Nothing tells the user which path they're on, how old the data is, or that these two paths can give contradictory answers about the same data.

This doc proposes a unified model: one flag (`--data-source`) that governs all read commands, defaults to live, falls back to local honestly when the network is down, and always tells you where your data came from.

## How It Works Today (Implementation Detail)

Printed CLIs that qualify for local search (determined by the profiler at generation time) get two commands: `sync` and `search`. Here's what the existing implementation does:

### `sync` command (`search.go.tmpl` → `sync.go.tmpl`)

The profiler (`internal/profiler/profiler.go`) walks the API spec at generation time and identifies list endpoints with pagination metadata. These become a hardcoded list of syncable resources baked into the printed CLI:

```go
// Generated — not discovered at runtime
func defaultSyncResources() []string {
    return []string{"pages", "databases", "users", "blocks"}
}
func syncResourcePath(resource string) string {
    paths := map[string]string{
        "pages":     "/v1/pages",
        "databases": "/v1/databases",
    }
    ...
}
```

When the user runs `sync`, the CLI paginates through each list endpoint using spec-derived cursor and limit parameter names, upserting every record into a local SQLite database at `~/.local/share/<cli-name>/data.db`. Pagination parameters, page size, and since-filter field names are all determined from the spec at generation time. Subsequent syncs are incremental — the store tracks `last_synced_at` per resource in a `sync_state` table and passes it as a `since` filter on the next sync.

### `search` command (`search.go.tmpl`)

The schema builder (`internal/generator/schema_builder.go`) decides which tables get FTS5 indexes: tables need 2+ text fields (title, name, description, body, etc.) and a gravity score >= 6 (based on endpoint count, field count, temporal fields, FK references). Qualifying tables get FTS5 virtual tables with porter stemming, plus INSERT/UPDATE/DELETE triggers to keep the index in sync.

The `search` command queries these FTS5 indexes. It supports `--type` to filter by resource and `--limit` for result count. It deduplicates across tables, filters out incomplete records, and outputs plain text or JSON.

### The generation gate

The profiler decides whether a printed CLI gets search at all (`internal/profiler/profiler.go`):

```go
p.NeedsSearch = len(listResources) >= 3 &&
    float64(searchEndpointCount)/float64(len(listResources)) < 0.5
```

Three or more list-capable resources, and fewer than half have dedicated API search endpoints. If the API already has good search coverage, these commands aren't generated.

### What's disconnected

These two commands exist in isolation from the rest of the CLI:

- **Regular commands** (`orders list`, `users get`) always hit the API live via the HTTP client. They have no awareness of the local SQLite database. There's no way to tell them to use local data, and no fallback if the network is down.
- **`search`** always hits local SQLite. It has no awareness of the live API. If the API has a search endpoint, the search command doesn't use it — it only queries the local FTS index.
- **`sync`** populates the local database but has no relationship to any other command. Nothing triggers it, nothing depends on it except `search`, and `search` doesn't check whether sync has been run recently or ever.

There is no shared model for "where does this data come from."

## Problem

1. **`search` requires an explicit `sync` first.** If the user hasn't synced, `search` opens the database, finds nothing, and errors: `"Run '<cli> sync' first to populate the local database."` There's no progressive disclosure — just a wall.

2. **No staleness signal.** The store tracks `last_synced_at` per resource, but the search command never reads it. After syncing once, every subsequent search returns results with zero indication of data age. A search 3 weeks post-sync looks identical to one 3 seconds post-sync. The user has no way to know they're looking at stale data without manually checking.

3. **User manages the cache.** The user is responsible for knowing that (a) `sync` exists, (b) they need to run it before `search`, and (c) they need to re-run it periodically to stay fresh. This is the data-warehouse mental model — fine for analysts, wrong for a CLI tool.

4. **Agents can't reason about freshness.** The `--json` output from `search` contains results but no metadata about data source, sync timestamp, or completeness. An agent calling `search` has no way to decide whether the results are trustworthy or 3 weeks old.

5. **No search endpoint utilization.** Even when the API has a search endpoint (which the profiler detects — `hasSearchEndpoint`), the `search` command never uses it. It always queries local FTS. The profiler uses search endpoint detection only to decide whether to *generate* the search command — not to route queries at runtime.

6. **Deletions are invisible.** `sync` upserts records but never removes them. If a record is deleted upstream, the local copy persists indefinitely. A search that returns a deleted record gives the user a false positive with no way to know.

7. **The live/local question applies to every command, not just search.** `orders list` and `users get` always hit the API. There's no `--data-source local` to use synced data, no fallback on network failure, and no consistency in how different commands resolve data.

## Motivation

Every read command in a printed CLI fetches data from a remote API. Every read command shares the same questions: what if the network is down? What if the user wants offline access? What if an agent wants fast, predictable reads without network latency?

Local FTS search makes this more visible because it *requires* local data, but the underlying design question — live vs local, and how to be honest about which one you're using — applies to the entire CLI.

## Design Principles

1. **Live by default.** Every command hits the API. This is what users expect from a CLI that talks to a service.
2. **No silent fallbacks.** The user always knows whether they're looking at live or cached data.
3. **No TTL.** The user or the network condition decides when to use local data, not a timer.
4. **No auto-sync.** Running a command doesn't secretly sync data in the background. `sync` is explicit.
5. **No read-through caching.** Regular commands do NOT write to the local database as a side effect. Local data is either complete (for synced resources) or absent — never a partial, query-shaped fragment that gives false confidence.
6. **Uniform model.** The same data source resolution logic applies to every read command, not just search.

## The Flag

One flag on the root command, three values:

```
--data-source auto|live|local    (default: auto)
```

This is a root-level persistent flag, inherited by all subcommands:

```bash
# Per-command
cli orders list --data-source local
cli search "refund" --data-source live

# Global for a session
cli --data-source local orders list
cli --data-source local search "refund"

# Configurable default
cli config set data-source local
```

### Behavior Matrix

| Mode | API reachable | API unreachable |
|------|--------------|-----------------|
| **`auto`** | Hit API live | Return local data + warn with sync timestamp |
| **`live`** | Hit API live | Error: `"API unreachable"` |
| **`local`** | Return local data, show sync age | Return local data, show sync age |

For all modes: if local data is needed and doesn't exist, error with `"No local data. Run 'sync' first."`.

### How this applies to different command types

| Command type | `auto` | `live` | `local` |
|-------------|--------|--------|---------|
| **`orders list`** | GET /orders | GET /orders | Query local SQLite `orders` table |
| **`orders get <id>`** | GET /orders/:id | GET /orders/:id | Query local SQLite by ID |
| **`search "query"`** | API search endpoint (if exists), else local FTS | API search endpoint (if exists), else error | Local FTS |
| **`orders create`** | POST /orders (always live) | POST /orders (always live) | POST /orders (always live) |

**Write commands (create, update, delete) are always live.** `--data-source` only affects reads. Mutations must hit the API — there's no local write path.

### Search-specific behavior

Search has one additional case: APIs without a search endpoint. The profiler knows this at generation time.

| API has search endpoint? | `auto` / `live` | `local` |
|-------------------------|-----------------|---------|
| **Yes** | Hit API search endpoint | Local FTS |
| **No** | Local FTS (with explanation: "This API has no search endpoint. Searching local data.") | Local FTS |

When the API has no search endpoint, `auto` and `live` both resolve to local FTS because there's nothing to hit live. The CLI says why.

## Resolution Flow

```
Any read command, --data-source <mode>:

1. mode = "live" (or "auto" default):
   → Hit API
   → Success? Return results, source = "live". Done.
   → Failed (network error) and mode = "auto"?
     → Local data exists for this resource?
       → Return local data + warn:
         "API unreachable. Showing cached data (synced <timestamp>)."
       → No local data?
         → Error: "API unreachable and no local data.
           Run 'sync' to enable offline access."
   → Failed and mode = "live"?
     → Error: "API unreachable."

2. mode = "local":
   → Local data exists? Return it + show sync age. Done.
   → No local data? Error: "No local data. Run 'sync' first."

3. Write commands (create, update, delete):
   → Always hit API. --data-source ignored.
```

## Output

All responses include provenance. Always. No exceptions.

**Human output (stderr):**

```
3 results (live)
```
```
3 results (cached, synced 2 hours ago)
```
```
⚠ API unreachable. 3 results (cached, synced 2 hours ago)
```
```
No search endpoint for this API. 3 results (cached, synced 2 hours ago)
```

**JSON output (--json):**

```json
{
  "results": [...],
  "meta": {
    "source": "live",
    "query": "refund"
  }
}
```
```json
{
  "results": [...],
  "meta": {
    "source": "local",
    "synced_at": "2026-04-04T10:30:00Z",
    "resource_type": "orders",
    "query": "refund",
    "reason": "api_unreachable"
  }
}
```

The `reason` field tells agents *why* local data was used:
- `api_unreachable` — network failure, auto mode fell back
- `no_search_endpoint` — API has no search, local FTS is the only option
- `user_requested` — user passed `--data-source local`

Agents use this to decide next steps without guessing.

## `sync` Command

Unchanged from today. Explicit, user-controlled:

```bash
sync                          # full sync, all resources
sync --resources orders       # just orders
sync --since 7d               # incremental, last week
sync --full                   # wipe and re-pull everything
sync --concurrency 8          # parallel workers
```

`sync` is the only way data enters local SQLite. It populates the data that `--data-source local` reads and that `auto` mode falls back to on network failure.

## Scope of Changes

### Generator (machine changes — every future CLI benefits)

| File | Change |
|------|--------|
| `root.go.tmpl` | Add `--data-source` persistent flag (auto\|live\|local) to root command. Pass through `rootFlags`. |
| `client.go.tmpl` | Add data source awareness. When `local`, read from store instead of HTTP. When `auto`, try HTTP first, fall back to store on network error. |
| `search.go.tmpl` | When `auto`/`live` and API has search endpoint: hit API. When `local` or no search endpoint: use FTS. Add provenance to output. |
| `store.go.tmpl` | Add `GetSyncTimestamps() map[string]time.Time` for provenance. Add generic query methods so non-search commands can read local data (e.g., `GetOrders(filters)`, `GetOrderByID(id)`). |
| `profiler.go` | Already detects `hasSearchEndpoint`. Additionally expose per-resource search endpoint paths and query parameter names in template data. |

### What does NOT change

| File | Why |
|------|-----|
| `sync.go.tmpl` | No changes. Sync stays explicit and user-initiated. |
| `config.go.tmpl` | No TTL config. Only addition: optional persistent `data-source` default. |
| Write command templates | Mutations always hit API. `--data-source` doesn't apply. |

### Profiler Data Additions

The search template needs to know at generation time:
- Does this API have a search endpoint? (already detected)
- What is the search endpoint path? (needs extraction)
- What is the query parameter name? (needs extraction — e.g., `q`, `query`, `search`)

For the broader local-read capability, the store needs typed query methods per resource. The schema builder already generates per-table FTS methods — this extends to basic filtered reads (by ID, by column filters that match CLI flags).

### Scoring / Verification

The scorecard should not penalize CLIs for lacking local search when the API has full search coverage. The `NeedsSearch` gate handles this at generation time, but verify scorer alignment.

## What This Does NOT Cover

- **Deletion detection.** Sync upserts but doesn't purge records deleted upstream. `sync --full` is the workaround (wipe and re-pull). Proper solution needs tombstone tracking or full-replace-per-resource semantics.
- **Write-through / cache invalidation.** Local data is read-only. Mutations go through the API. No automatic re-sync after writes.
- **Real-time / webhooks.** No push-based sync.
- **Read-through caching.** Regular commands do NOT populate the local database. Intentional — partial data is worse than no data.
- **Per-resource data source config.** V1 uses a single `--data-source` for all resources. Per-resource overrides (e.g., "orders always live, users can be local") are a reasonable future extension.

## Success Criteria

1. Every read command defaults to live API calls — no behavior change from what users expect today.
2. `--data-source local` works for any read command after `sync`, not just search.
3. `--data-source auto` falls back to local on network failure with clear warning and sync timestamp.
4. Provenance metadata (source, sync age, fallback reason) appears on every response — human and JSON.
5. Write commands always hit the API regardless of `--data-source`.
6. `sync` command unchanged. Remains the only path to populate local data.
7. No TTL. No auto-sync. No read-through caching. No silent fallbacks.
