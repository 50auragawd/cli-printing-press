---
title: "fix: Generator improvements from postman-explore retro"
type: fix
status: active
date: 2026-03-30
origin: docs/retros/2026-03-30-postman-explore-retro.md
---

# fix: Generator improvements from postman-explore retro

## Overview

The postman-explore CLI generation surfaced 8 systemic issues in the printing-press generator. This plan addresses the 4 highest-priority work units from the retro — all targeting generator templates so future CLIs come out stronger with less manual rework.

## Problem Frame

Every CLI generation session currently requires 5-7 manual edits to generated code. The most expensive are sync rewrites (offset pagination, response envelopes), missing batch store methods, and DB path mismatches. The spec structs already carry the metadata needed to fix most of these — the templates just don't use it. (see origin: docs/retros/2026-03-30-postman-explore-retro.md)

## Requirements Trace

- R1. Sync template branches on `Pagination.Type` (offset/cursor/page_token/none) instead of hardcoding cursor
- R2. Sync template uses `Endpoint.ResponsePath` or `extractPageItems` for envelope unwrapping
- R3. Store template emits `UpsertBatch<Entity>()` for high-gravity tables
- R4. Single `defaultDBPath()` in helpers.go.tmpl, referenced by all store-consuming templates
- R5. Proxy-envelope `serviceForPath()` correctly reads `.ProxyRoutes` — verify data propagation from spec to template context
- R6. No regressions for standard REST APIs (cursor-paginated, no proxy, no envelope)

## Scope Boundaries

- No changes to the profiler, spec parser, or OpenAPI parser — they already produce the right metadata
- No new CLI commands (e.g., `printing-press polish`) — template-level fixes only
- No changes to scorecard, dogfood, or verify tools
- Dead code conditional emission (retro finding #7) is deferred — low fallback cost, higher complexity
- Top-level command alias generation (retro finding #4) is deferred — high value but needs separate design

## Context & Research

### Relevant Code and Patterns

- **Spec pagination struct** (`internal/spec/spec.go:79-85`): `Pagination{Type, LimitParam, CursorParam, NextCursorPath, HasMoreField}` — already populated by parser
- **Profiler pagination** (`internal/profiler/profiler.go:39-45`): Aggregates per-endpoint pagination into API-wide pattern with `mostCommon()` fallbacks
- **Sync template** (`internal/generator/templates/sync.go.tmpl`): `determinePaginationDefaults()` at line 298 hardcodes `cursorParam: "after"` — ignores spec metadata
- **Command endpoint template** (`internal/generator/templates/command_endpoint.go.tmpl:62-69`): Correctly passes `Endpoint.Pagination.*` to `paginatedGet()` — proves the pattern works for individual commands, just missing from bulk sync
- **Store template** (`internal/generator/templates/store.go.tmpl`): Already generates per-entity `Upsert<Entity>()` (lines 262-306) and `Search<Entity>()` (lines 341-372) via `{{range .Tables}}`. Missing: `UpsertBatch<Entity>()`
- **Client template** (`internal/generator/templates/client.go.tmpl`): Already has `{{range $prefix, $svc := .ProxyRoutes}}` for proxy routing — template logic appears correct
- **DB path fragmentation**: 5 templates define paths independently; `channel_workflow.go.tmpl:200` uses `~/.config` while others use `~/.local/share`
- **Sync template data**: Receives `SyncableResources []string`, `SearchableFields map[string][]string`, `Tables []TableDef` — but NOT per-resource pagination metadata

### Institutional Learnings

- Dead code detection uses `strings.Count(allContent, name+"(") < 2` — any new template helper must be called at least once or it flags (see origin: docs/solutions/logic-errors/scorecard-accuracy-broadened-pattern-matching-2026-03-27.md)
- `spec_source: sniffed` gates sniffed-API behaviors (rate limiting defaults). Could also gate conservative pagination/envelope strategies (see origin: docs/solutions/best-practices/adaptive-rate-limiting-sniffed-apis.md)
- Workflow commands detected by `store.Open`/`store.New` usage; insight commands need aggregation patterns. Template changes must preserve these scoring signals (see origin: docs/solutions/best-practices/steinberger-scorecard-scoring-architecture-2026-03-27.md)

## Key Technical Decisions

- **Pagination: branch in template, not in profiler** — The profiler already computes per-endpoint pagination. The fix is in the sync template reading what's already there, not in computing new metadata. Rationale: the command_endpoint template already does this successfully.
- **DB path: single definition in helpers, not per-template** — Rather than fixing each template's inline path, emit one `defaultDBPath()` in helpers and have all templates call it. Rationale: single source of truth, matches how `newTabWriter()` and other shared helpers work.
- **Batch upsert: template addition, not schema builder change** — The schema builder already provides `TableDef` with all needed metadata. The store template just needs a new `{{range}}` block for batch methods.
- **Proxy route investigation before fix** — The client template's proxy logic looks correct on paper. Before changing it, verify whether `spec.ProxyRoutes` is actually populated when generating from a catalog entry with `proxy_routes`. The issue may be data propagation, not template logic.

## Open Questions

### Resolved During Planning

- **Q: Does the sync template have access to per-resource pagination metadata?** — Not directly. `SyncableResources` is `[]string` (resource names only). The template would need either: (a) the full `Pagination` struct per resource passed in, or (b) access to `spec.Resources[name].Endpoints[...].Pagination`. Option (a) requires a small change to `generator.go` to build a richer sync data struct.
- **Q: Does store.go.tmpl already generate per-entity methods?** — Yes, `Upsert<Entity>()` and `Search<Entity>()` are generated. Only `UpsertBatch<Entity>()` is missing.

### Deferred to Implementation

- **Q: What's the exact failure mode for ProxyRoutes?** — Need to trace `spec.ProxyRoutes` from YAML parse through generator context to template render. May be a nil map vs empty map issue, or a field name mismatch.
- **Q: Should `defaultSyncResources()` be derived from spec entities?** — Currently hardcoded placeholder. Could be generated from high-gravity table names, but may produce wrong results for APIs with many low-value resources.

## Implementation Units

- [ ] **Unit 1: Consolidate DB path into helpers.go.tmpl**

  **Goal:** Single `defaultDBPath()` function, all store-consuming templates reference it.

  **Requirements:** R4

  **Dependencies:** None

  **Files:**
  - Modify: `internal/generator/templates/helpers.go.tmpl`
  - Modify: `internal/generator/templates/sync.go.tmpl`
  - Modify: `internal/generator/templates/search.go.tmpl`
  - Modify: `internal/generator/templates/analytics.go.tmpl`
  - Modify: `internal/generator/templates/channel_workflow.go.tmpl`
  - Modify: `internal/generator/templates/mcp_tools.go.tmpl`
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - Add `defaultDBPath()` to helpers.go.tmpl returning `filepath.Join(home, ".local", "share", "<cli-name>", "data.db")`
  - Replace all 5 inline `filepath.Join(home, ...)` DB path constructions with `defaultDBPath()`
  - Remove the existing `defaultDBPath()` from `channel_workflow.go.tmpl`

  **Patterns to follow:**
  - Existing shared helpers in `helpers.go.tmpl` (e.g., `newTabWriter()`, `truncate()`)

  **Test scenarios:**
  - Happy path: Generate a CLI, grep for `UserHomeDir` in `internal/cli/` — only one definition of DB path exists
  - Happy path: All store-consuming commands (`sync`, `search`, `analytics`, `workflow`) use the same path
  - Edge case: Generated `defaultDBPath()` uses the correct CLI name from spec

  **Verification:**
  - `go build` succeeds
  - `grep -r "filepath.Join.*local.*share" internal/cli/` returns only the helpers.go definition

- [ ] **Unit 2: Investigate and fix proxy route data propagation**

  **Goal:** Verify `spec.ProxyRoutes` flows from catalog/spec parse through generator to client template. Fix if broken.

  **Requirements:** R5

  **Dependencies:** None (independent of Unit 1)

  **Files:**
  - Read: `internal/spec/spec.go` (ProxyRoutes field definition)
  - Read: `internal/spec/reader.go` or `internal/openapi/parser.go` (where ProxyRoutes is populated)
  - Read: `internal/generator/generator.go` (where spec is passed to template)
  - Read: `internal/generator/templates/client.go.tmpl` (template logic)
  - Modify: whichever file has the propagation gap (determined during investigation)
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - Trace `ProxyRoutes` from spec YAML field through parse, into `APISpec` struct, through generator context, to template rendering
  - The template logic (`{{range $prefix, $svc := .ProxyRoutes}}`) looks correct — suspect the data isn't reaching it
  - Check: is the YAML field `proxy_routes` or `proxyRoutes`? Does the parser handle the field name correctly?
  - Check: does the generator pass `*spec.APISpec` directly or copy fields into a new struct (which could drop ProxyRoutes)?

  **Patterns to follow:**
  - How other spec fields (e.g., `ClientPattern`, `Auth`) propagate to templates — follow the same path

  **Test scenarios:**
  - Happy path: Generate from postman-explore spec with `proxy_routes` — `serviceForPath("/search-all")` returns `"search"` in generated code
  - Happy path: Generate from postman-explore spec — `serviceForPath("/v1/api/team")` returns `"publishing"`
  - Negative test: Generate from a standard REST spec without proxy_routes — no `serviceForPath` function emitted, standard `do()` method used

  **Verification:**
  - Generated `client.go` for proxy-envelope API contains path-specific routing
  - Generated `client.go` for standard REST API does not contain proxy routing

- [ ] **Unit 3: Pagination-aware sync template**

  **Goal:** `sync.go.tmpl` reads per-resource pagination metadata and emits correct pagination logic per endpoint type.

  **Requirements:** R1, R6

  **Dependencies:** None (independent)

  **Files:**
  - Modify: `internal/generator/generator.go` (pass pagination metadata to sync template)
  - Modify: `internal/generator/templates/sync.go.tmpl` (branch on pagination type)
  - Modify: `internal/spec/spec.go` (possibly — check if `SyncableResources` needs enrichment)
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - Currently `SyncableResources` is `[]string`. Enrich it to carry pagination metadata per resource. Either:
    - (a) Change to `[]SyncResource{Name string, Pagination spec.Pagination}` — cleaner
    - (b) Pass full `spec.Resources` map alongside and look up by name in template — more fragile
  - Option (a) is preferred. Add a `SyncResource` struct in generator.go and populate it from `spec.Resources`
  - In `sync.go.tmpl`, replace hardcoded `determinePaginationDefaults()` with a per-resource branch:
    - `Type == "offset"`: use offset+limit loop, increment offset by page size
    - `Type == "cursor"`: use current cursor-based logic (existing behavior, stays default)
    - `Type == "page_token"`: similar to cursor but with token-specific field names
    - `Type == ""` or no pagination detected: fetch once, no loop
  - The `command_endpoint.go.tmpl` already does this for individual commands — mirror its approach

  **Patterns to follow:**
  - `command_endpoint.go.tmpl:62-69` — how it passes `Endpoint.Pagination.*` to `paginatedGet()`
  - `profiler.go:39-45` — `PaginationProfile` struct and `mostCommon()` aggregation

  **Test scenarios:**
  - Happy path: Generate from a spec with offset-paginated endpoints — sync uses `offset` param and terminates after last page
  - Happy path: Generate from a spec with cursor-paginated endpoints — sync uses `after`/cursor param (existing behavior preserved)
  - Edge case: Generate for an endpoint with no pagination metadata — sync fetches once, no loop
  - Edge case: Mixed API with some cursor and some offset endpoints — each resource uses its own pagination type
  - Negative test: Generate from Stripe-like spec (cursor) — sync still works as before (no regression)

  **Verification:**
  - `go build` succeeds
  - Generated sync.go for postman-explore-like spec contains offset-based pagination
  - Generated sync.go for standard REST spec contains cursor-based pagination

- [ ] **Unit 4: Response envelope unwrapping in sync**

  **Goal:** Sync template uses spec's `ResponsePath` metadata to unwrap response envelopes instead of relying on heuristic.

  **Requirements:** R2

  **Dependencies:** Unit 3 (sync template rework — changes to the same template file)

  **Files:**
  - Modify: `internal/generator/templates/sync.go.tmpl`
  - Modify: `internal/generator/generator.go` (if ResponsePath needs to be passed to sync context)
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - If `SyncResource` struct from Unit 3 includes a `ResponsePath` field (e.g., `"data"`), the sync template can emit targeted unwrap code instead of calling `extractPageItems`
  - When `ResponsePath` is set: `json.Unmarshal(data, &envelope)` then `items = envelope[responsePath]`
  - When `ResponsePath` is empty: fall through to existing `extractPageItems` heuristic
  - This follows the same pattern as the rate limiter: spec metadata gates behavior, heuristic as fallback

  **Patterns to follow:**
  - `extractPageItems()` in current sync template — the heuristic to fall back to
  - `spec_source` gating in rate limiter — metadata-driven behavior with fallback

  **Test scenarios:**
  - Happy path: Generate from spec where endpoints declare `response_path: "data"` — sync unwraps `{"data": [...]}` correctly
  - Happy path: Generate from spec without response_path — sync falls back to `extractPageItems` heuristic
  - Edge case: Different endpoints have different response paths — each sync function uses its own
  - Negative test: Generate from spec returning direct arrays — no unwrap code emitted

  **Verification:**
  - Sync for postman-explore-like API parses `{"data": [...]}` without manual `unwrapDataArray()` helper

- [ ] **Unit 5: Emit UpsertBatch per entity in store template**

  **Goal:** `store.go.tmpl` generates `UpsertBatch<Entity>()` methods for high-gravity tables, so Claude doesn't write 24 methods from scratch.

  **Requirements:** R3

  **Dependencies:** None (independent of sync changes)

  **Files:**
  - Modify: `internal/generator/templates/store.go.tmpl`
  - Test: `internal/generator/generator_test.go`

  **Approach:**
  - The template already iterates `{{range .Tables}}` to emit `Upsert<Entity>()` and `Search<Entity>()`. Add a third block for `UpsertBatch<Entity>()` using the same table metadata
  - The batch method should: begin transaction, iterate items, call `Upsert<Entity>Tx()` per item (extract the existing upsert logic into a `Tx` variant), commit
  - Only emit for tables with gravity >= 6 (same threshold as existing per-entity methods)
  - Also emit the FTS update within the batch transaction (matching what the existing `Upsert<Entity>()` does for FTS)

  **Patterns to follow:**
  - Existing `Upsert<Entity>()` in store.go.tmpl (lines 262-306) — same field extraction, same FTS update
  - Existing generic `UpsertBatch()` in store.go.tmpl — same transaction pattern but with typed fields

  **Test scenarios:**
  - Happy path: Generate a CLI with high-gravity entities — `store.go` contains `UpsertCollectionBatch()` method
  - Happy path: Batch method uses entity-specific FTS table (not `resources_fts`)
  - Edge case: Low-gravity entity (< 6) — no batch method emitted, uses generic `UpsertBatch()`
  - Negative test: Generate from minimal spec with no high-gravity entities — only generic batch method exists

  **Verification:**
  - Generated `store.go` for postman-explore-like spec contains `UpsertCollectionBatch`, `UpsertCategoryBatch`, `UpsertTeamBatch`
  - Each batch method updates entity-specific FTS index, not `resources_fts`

## System-Wide Impact

- **Sync template changes (Units 3-4)** touch the most complex generated file. Changes must preserve the existing worker pool, progress reporting, and sync state tracking. The new pagination branching adds complexity but is localized to `determinePaginationDefaults()` and `syncResource()`
- **Generator context changes (Unit 3)** change the data struct passed to the sync template. Other templates that receive the same struct must not break — verify that `SyncableResources` consumers handle the type change
- **Store template changes (Unit 5)** add methods but don't modify existing ones. Low regression risk. The `UpsertBatch<Entity>` methods must follow the same FTS update pattern as `Upsert<Entity>` to avoid search index divergence
- **Scorecard interaction**: New batch methods will be detected as store-using code, maintaining or improving DataPipelineIntegrity scores. Dead code risk is low since batch methods will be called by the sync template
- **Unchanged invariants**: The generic `resources` table, `resources_fts`, `UpsertBatch()`, `Search()`, and all other generic store methods remain unchanged. The entity-specific methods are additive

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Sync template changes break existing cursor-paginated CLIs | Unit 3 explicitly preserves cursor as default. Test against Stripe-like spec |
| `SyncResource` struct change breaks other template consumers | `SyncableResources` is only used by sync.go.tmpl — verify no other templates reference it |
| Batch upsert FTS updates diverge from single upsert | Extract shared FTS logic into template helper, or keep the two in sync via code review |
| ProxyRoutes investigation (Unit 2) finds no bug — template was already correct | If so, Unit 2 becomes a test-only unit (add regression test, no code change). Low wasted effort |

## Sources & References

- **Origin document:** [docs/retros/2026-03-30-postman-explore-retro.md](docs/retros/2026-03-30-postman-explore-retro.md)
- Generator templates: `internal/generator/templates/` (35 files)
- Spec types: `internal/spec/spec.go` (Pagination struct at lines 79-85)
- Profiler: `internal/profiler/profiler.go` (PaginationProfile at lines 39-45)
- Schema builder: `internal/generator/schema_builder.go` (BuildSchema, data gravity scoring)
- Scorecard accuracy learning: `docs/solutions/logic-errors/scorecard-accuracy-broadened-pattern-matching-2026-03-27.md`
- Adaptive rate limiting learning: `docs/solutions/best-practices/adaptive-rate-limiting-sniffed-apis.md`
