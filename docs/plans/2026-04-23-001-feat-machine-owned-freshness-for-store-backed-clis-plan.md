---
title: "feat: Machine-Owned Freshness for Store-Backed Printed CLIs"
type: feat
status: active
date: 2026-04-23
---

# feat: Machine-Owned Freshness for Store-Backed Printed CLIs

## Overview

The Product Hunt follow-up PR surfaced a useful distinction:

1. Some of the work is **Product Hunt specific** and should stay in that printed CLI.
2. Some of the work is a **machine gap**: store-backed printed CLIs need a better default story for freshness ownership, cold-start behavior, and agent-facing observability.

This plan focuses on the machine. The goal is not to make every printed CLI behave like Product Hunt. The immediate goal is narrower: make the existing Printing Press freshness machinery usable by generated read commands and explicitly registered custom store-backed commands, so future printed CLIs start with a better freshness baseline instead of reimplementing small wrappers one CLI at a time.

The core insight is broad, but v1 should stay conservative: **when a printed CLI has a local store and a command path has explicitly opted into machine-managed freshness, the CLI should own the bounded freshness check before serving that command.** API-specific enrichment and backfill tiers may still require per-CLI work, and machine-wide scoring or new schema concepts should wait until the helper contract is proven in more than one printed CLI.

## Problem Frame

The Product Hunt PR tried to make `producthunt-pp-cli` "self-warming" by adding:

- automatic Atom sync before read commands
- `search --enrich` to top up thin local results from GraphQL
- explicit GraphQL backfill and resume flows

That direction has merit. Agents and downstream tools should not have to know "run sync every 24h, then query the store." They should be able to call the semantic read command and let the CLI decide whether the store is fresh enough.

But the PR also showed why this must be separated carefully:

- **The machine already has part of this capability.** `spec.Cache`, `cliutil_freshness.go.tmpl`, `auto_refresh.go.tmpl`, and the root pre-run hook already provide an opt-in stale-cache refresh path for generated store-backed CLIs.
- **The current machine capability is too narrow and too invisible.** It is positioned as "cache auto-refresh," not as a first-class agent-facing freshness contract for store-backed read commands.
- **Custom hand-authored commands sit outside the generic story.** Product Hunt had to invent `autoWarm()` and `AutoSyncMeta` because the generated freshness helpers are not exposed as an ergonomic extension seam for custom local-store commands.
- **The machine does not expose a clear coverage registry for custom commands.** Generated commands have `readCommandResources`, but hand-authored local-store commands do not have a canonical way to register "this command path reads these resources and may refresh them."
- **Generated docs do not yet express this nuance.** The machine can produce or permit claims like "self-warming" without specifying which commands are warmed, what source they use, and what happens on explicit live-only paths.

If we do nothing at the machine level, future store-backed CLIs will keep rediscovering the same pattern in custom code:

- pre-read freshness check
- bounded fail-soft refresh
- opt-out flag/env
- stale-vs-never-synced distinction
- hidden coupling between read commands and sync/backfill layers
- missing or inconsistent JSON metadata about the refresh decision

That is exactly the kind of repeated work the Printing Press should absorb.

## Why This Matters Broadly

This pattern is valuable beyond Product Hunt for any printed CLI that:

- syncs API data into SQLite for offline reads, search, or insights
- derives agent-useful views from historical snapshots
- uses a local store as the main interface instead of raw passthrough endpoints
- has cheap incremental refresh but expensive historical hydration

Broad benefits, if the v1 helper contract proves itself across more than Product Hunt:

- **Better first-run usability.** Agents can issue the meaningful read command instead of learning a vendor-specific "sync first" ritual.
- **Less duplicated orchestration in downstream tools.** Skills, MCP hosts, and scripts no longer each invent their own stale-store policy.
- **More honest CLI contracts.** A printed CLI can explicitly say what it owns: freshness of current cache, not necessarily full historical completeness.
- **Safer defaults for agents.** A bounded fail-soft refresh is better than empty-store surprises or unbounded background sync.
- **A cleaner machine/printed split.** The machine provides bounded freshness for declared command paths; individual printed CLIs can layer API-specific enrichment or backfill when justified.

## Requirements Trace

- R1. Treat store-backed read freshness for explicitly registered command paths as a first-class machine capability, not an ad hoc per-CLI trick.
- R2. Build on the existing `spec.Cache` / auto-refresh machinery rather than introducing a second competing abstraction.
- R3. Expose a reusable extension seam so hand-authored read commands in printed CLIs can participate in the same freshness behavior and metadata contract as generated commands.
- R4. Distinguish baseline freshness from optional historical backfill or enrichment. The machine must not overclaim that a CLI is "fully warm" when only declared command paths are covered.
- R5. Preserve command semantics for source selection. `--data-source live` must not silently mutate the local store first; `--data-source local` must not refresh; `--data-source auto` may perform bounded refresh when the command path is registered.
- R6. Keep freshness opt-out separate from source selection. Env opt-out and any future `--no-auto-refresh` disable the pre-read freshness hook while leaving `--data-source` semantics intact.
- R7. Make freshness decisions observable to agents in a stable JSON shape when JSON output uses the generated provenance envelope.
- R8. Update generated README/SKILL templates so the machine accurately describes freshness ownership, opt-outs, and limitations for store-backed CLIs.
- R9. Verify the capability based on actual runtime behavior for CLIs with a machine-testable freshness fixture. Do not score generic cross-CLI freshness ownership from fixture-only evidence.

## Scope Boundaries

- Not generating Product Hunt's GraphQL enrichment and backfill stack for every CLI.
- Not assuming every store-backed CLI should auto-refresh by default. This remains opt-in and context-sensitive.
- Not defining a universal "enrichment API" abstraction. Topic-aware top-up, budget handling, and OAuth registration are API-specific.
- Not forcing every historical or insight command to self-hydrate. Some commands should still require explicit sync/backfill if the cost profile is high.
- Not changing the machine to claim completeness it cannot provide. "Fresh enough to serve current reads" is different from "historically complete for 30 days."
- Not introducing a general read-model schema in v1. V1 uses command-path coverage because the generator already has that concept.
- Not adding new scorecard weight in v1. Scoring can follow after at least two printed CLIs use the helper contract and verify can test it end to end.

## Separation: Product Hunt Specific vs Machine Generalizable

### Product Hunt-Specific Work

These should stay in the printed CLI unless a later cross-CLI pattern proves they generalize:

- Atom as the cheap default read surface and GraphQL as the expensive historical surface
- OAuth app registration flow against Product Hunt's `client_credentials` token endpoint
- Product Hunt budget tracking and resume behavior around GraphQL complexity limits
- Topic-aware `search --enrich` heuristics, including time window and client-side filtering
- Whether GraphQL backfill should write only `posts`, also write synthetic `snapshots`, or both
- Product Hunt command-specific semantics such as `today`, `trend`, and snapshot-derived ranking history

### Machine-Generalizable Work

These should move into the Printing Press because they help many future store-backed CLIs:

- A first-class freshness contract for explicitly registered store-backed read command paths
- Generated helpers for deciding whether a read path should refresh before serving
- Stable agent-facing metadata describing the freshness decision
- Clear separation between fresh cache, stale cache, no store, source selection, and freshness opt-out
- A command-path registry so the machine knows which generated and custom commands are covered by which resources
- Template support so the generated README and SKILL describe the contract accurately
- Runtime verify coverage for freshness-owned read paths when the CLI declares a machine-testable fixture

## Existing Machine Surface to Reuse

The machine already contains the right seed crystals:

- `internal/spec/spec.go` - `CacheConfig` already models stale-after, refresh timeout, per-resource thresholds, and env opt-out.
- `internal/generator/templates/cliutil_freshness.go.tmpl` - already computes freshness decisions from `sync_state`.
- `internal/generator/templates/auto_refresh.go.tmpl` - already performs bounded fail-soft refreshes for generated read commands.
- `internal/generator/templates/root.go.tmpl` - already hooks auto-refresh into persistent pre-run for generated read commands.

The plan should therefore **extend and rename the mental model**, not replace it:

- "cache auto-refresh" should become "machine-owned read freshness for registered store-backed command paths"
- the helper surface should be reusable by custom commands
- the docs and verification layer should understand what the capability actually guarantees

## Key Technical Decisions

- **KTD-1: Build on `spec.Cache`; do not invent `spec.SelfWarming`.** The current machine already has a freshness abstraction. Renaming the concept at the documentation/review level is cheaper and safer than introducing a parallel config block with overlapping semantics.

- **KTD-2: The machine capability is baseline freshness, not generic enrichment/backfill.** Auto-refresh is the default machine leg for registered command paths. Narrow enrichment and deep backfill remain optional printed-CLI extensions.

- **KTD-3: V1 freshness coverage is keyed by command path.** The existing machine already maps generated command paths to resource names in `readCommandResources`. V1 extends that registry so custom commands can opt in with explicit `command path -> resources` entries. "Read model" remains a design note, not a spec concept, until a second printed CLI proves command-path coverage is insufficient.

- **KTD-4: Explicit live-mode commands must not silently mutate local state.** The generated freshness hook must respect bypass semantics consistently. Product Hunt's `today --live` regression is exactly what the machine should prevent.

- **KTD-5: Freshness metadata extends the existing generated provenance envelope.** The current data-layer envelope is `{"results": <payload>, "meta": {...}}`; freshness metadata lives at `meta.freshness`. Commands that still emit bare arrays or custom JSON must opt into a generated envelope helper before claiming JSON freshness metadata. V1 does not silently wrap arbitrary custom outputs because that would be an output contract change.

- **KTD-6: Hand-authored commands need the same helper contract as generated commands, but only when they opt in.** The machine should emit a small reusable helper surface that custom commands can call without rewriting freshness logic, output metadata plumbing, and bypass checks. Custom commands that do not register a command path remain outside freshness coverage and outside freshness docs/verify claims.

## Open Questions

### Resolve During Planning / Design

- Should generated bypass control be a dedicated `--no-auto-refresh` flag, continue as env-only opt-out, or both?
- Should the first pilot require two CLIs before adding scorecard treatment, or is Product Hunt plus one generated fixture enough?

### Defer to Implementation

- Exact registration shape for custom command paths: generated map extension, spec-side `extra_commands`, or a small hand-authored Go registration helper
- Whether verify should simulate stale-store transitions via fixture seeding or manipulated `sync_state`
- Whether a later scorecard change belongs in a new dimension or the existing data-layer rubric after adoption is demonstrated

## Runtime Semantics Matrix

V1 must make the first-run and bypass behavior explicit so implementers do not infer different contracts.

| State | `--data-source auto` | `--data-source local` | `--data-source live` |
|-------|----------------------|-----------------------|----------------------|
| Fresh store | Serve local data; `meta.freshness.decision = "fresh"` | Serve local data; no refresh | Bypass freshness hook; call live path only |
| Stale `sync_state` | Run bounded refresh, then serve local data; if refresh fails, serve stale data with warning/meta error | Serve stale local data; no refresh | Bypass freshness hook; call live path only |
| Missing DB or missing `sync_state` | Run bounded initial refresh only when the command path is covered and a sync path exists; otherwise return not-hydrated metadata | Return not-hydrated/local-empty behavior; no refresh | Bypass freshness hook; call live path only |
| Zero-row store after refresh | Return normal empty result plus freshness metadata | Return normal empty local result | Return live result if the command has a live path |
| Env/future flag freshness opt-out | Skip freshness hook and serve according to command's existing `auto` behavior | Same as local | Same as live |

The important distinction: source selection chooses where the command reads from; freshness opt-out only disables the pre-read refresh hook.

## Implementation Units

- [x] **Unit 1: Reframe the freshness model around command-path coverage**

**Goal:** Make the existing `spec.Cache` machinery the canonical machine abstraction for store-backed read freshness, with command-path coverage as the v1 source of truth.

**Requirements:** R1, R2, R4, R5, R6

**Files:**
- Modify: `internal/spec/spec.go`
- Modify: `internal/spec/spec_test.go`
- Modify: `docs/PIPELINE.md`
- Modify: `AGENTS.md`

**Approach:**
- Keep the current `cache.enabled`, stale-after, refresh-timeout, per-resource thresholds, and env opt-out semantics intact.
- Define command-path coverage as the v1 authoritative unit: each covered command path maps to the resource names the freshness hook may refresh.
- Generated commands continue to populate coverage from existing `SyncableResources`.
- Custom commands can opt in only through an explicit registry entry. The plan should pick the least invasive implementation shape during this unit: either an extension of generated `readCommandResources`, a small `RegisterFreshnessCommand(path, resources...)` helper, or `extra_commands` metadata if that already fits the repo's patterns.
- Document that "read model" is deferred terminology. V1 avoids a new read-model schema until command-path coverage proves insufficient.
- Separate source-selection semantics from freshness opt-out semantics:
- `--data-source auto`: covered command paths may perform bounded refresh.
- `--data-source live`: no pre-read local-store mutation.
- `--data-source local`: no refresh; serve local data or report not hydrated.
- env opt-out / future `--no-auto-refresh`: disables the pre-read freshness hook while preserving the selected data source.

**Test scenarios:**
- Spec with current `cache.enabled` shape still validates unchanged.
- Generated command-path coverage validates for syncable resources.
- Custom command coverage validates only when each referenced resource is syncable or otherwise explicitly declared as refreshable.
- Coverage declaration for an unknown command path or unknown resource fails with a clear validation error.
- `--data-source` semantics and freshness opt-out semantics are documented as separate controls.
- Docs review: `AGENTS.md` and `docs/PIPELINE.md` describe freshness ownership without implying universal historical completeness.

- [x] **Unit 2: Add the missing custom-command helper surface**

**Goal:** Provide a shared helper contract so custom hand-authored commands can participate in the same freshness and metadata behavior as generated commands.

**Requirements:** R3, R5, R6

**Files:**
- Modify: `internal/generator/templates/cliutil_freshness.go.tmpl`
- Modify: `internal/generator/templates/cliutil_freshness_test.go.tmpl`
- Modify: `internal/generator/templates/auto_refresh.go.tmpl`
- Modify: `internal/generator/templates/root.go.tmpl`
- Modify: `internal/generator/generator.go`

**Approach:**
- Preserve the existing decision helper where possible. This unit should add the missing reusable pieces rather than restructure the full helper layer.
- Add or expose a bypass-aware refresh wrapper that custom commands can call with a command path or resource list.
- Add a generated metadata type with stable fields such as `decision`, `ran`, `reason`, `resources`, `elapsed_ms`, `error`, and `source`.
- Add a JSON metadata attachment helper for commands using the generated provenance envelope. The helper writes freshness metadata at `meta.freshness`.
- Commands that emit bare arrays or custom JSON must opt into the generated envelope helper before claiming JSON freshness metadata. V1 should not silently wrap arbitrary custom outputs.
- Make the helper callable from hand-authored commands in printed CLIs without requiring them to duplicate store-open, freshness-decision, and metadata-plumbing logic.
- Enforce bypass semantics: explicit live-only modes skip freshness-triggered mutation.
- Keep generated command behavior backward compatible for CLIs already using `cache.enabled`.

**Test scenarios:**
- Generated read command in `data-source auto` mode refreshes stale data and attaches freshness metadata in JSON mode.
- Generated read command in `data-source local` mode does not refresh and reports the correct bypass/meta reason.
- Generated read command in `data-source live` mode does not refresh or mutate the store first.
- Custom-command fixture can call the emitted helper and receives the same metadata shape as a generated command.
- Bare-array custom command fixture does not claim `meta.freshness` until it opts into the generated envelope helper.
- Refresh failure becomes a warning plus stale serve, not a hard command failure.

- [x] **Unit 3: Teach generated docs to describe freshness honestly**

**Goal:** Ensure the machine-generated README/SKILL describe freshness ownership, opt-outs, and limits accurately for store-backed CLIs.

**Requirements:** R4, R7

**Files:**
- Modify: `internal/generator/templates/readme.md.tmpl`
- Modify: `internal/generator/templates/skill.md.tmpl`

**Approach:**
- Add conditional doc language for store-backed CLIs with machine-owned freshness:
  - what the CLI refreshes automatically
  - when it will not refresh
  - what "fresh" means
  - what still requires explicit sync/backfill
- Prevent boilerplate claims that imply universal warmth or historical completeness.
- Generate docs from the same command-path coverage registry used by runtime behavior.
- Defer `skills/printing-press*.md` review-pipeline automation until the freshness contract and JSON metadata shape are stable in at least one generated CLI and one custom-command CLI.

**Test scenarios:**
- Store-backed CLI with freshness enabled gets README/SKILL text that mentions bounded auto-refresh and opt-out paths.
- CLI without freshness enabled does not receive self-warming language.
- CLI with optional explicit backfill documents it as additive, not part of the baseline freshness contract.
- Generated docs list covered command paths or command families derived from the registry and do not claim coverage for unregistered custom commands.

- [x] **Unit 4: Verify machine-owned freshness behavior**

**Goal:** Make freshness behavior a tested machine capability rather than a template best-effort.

**Requirements:** R9

**Files:**
- Modify: `internal/pipeline/verify.go`
- Modify: `internal/pipeline/runtime.go`
- Modify: relevant `internal/pipeline/*_test.go` files covering runtime verify and stale-store setup
- Modify: any relevant verify fixture builders or test helpers for store-backed CLIs

**Approach:**
- Add verify coverage for freshness-owned read paths:
  - stale store triggers bounded refresh
  - explicit live/local bypass works
  - JSON mode exposes the freshness decision
- Add a runtime verify fixture contract for freshness-aware CLIs. The fixture must declare how to seed the store, how to mark resources stale, and which covered command path should be exercised.
- Thread freshness verify results into the report contract so scorecard can consume them later.
- Do not add scorecard weight in v1. The first proof target is runtime behavior, not a new quality grade.
- Split the proof strategy:
- deterministic generator/unit tests prove helper mechanics
- runtime verify proves behavior only for CLIs that declare a machine-testable freshness fixture
- later scorecard work can use these runtime results after the contract is adopted by at least two printed CLIs

**Test scenarios:**
- Verify fixture with stale `sync_state` proves a read command refreshes before serving.
- Verify fixture with `data-source live` proves no pre-read store mutation occurs.
- Verify fixture with `data-source local` proves no refresh occurs and the command reports local/not-hydrated behavior.
- Verify fixture with refresh failure proves stale serve plus warning behavior.
- Verify report distinguishes CLIs that implement the freshness contract from those that only have a local store.

## Printed-CLI Follow-On Work After Machine Changes

These are not machine units, but they should be tracked explicitly so we do not blur the boundary:

### Product Hunt follow-on fixes

- Fix `today --live` so explicit live mode never performs implicit store mutation first.
- Fix `auth logout` to clear all saved OAuth credentials, not only access tokens.
- Either attach freshness metadata consistently or stop claiming it in docs until it exists.
- Re-scope `search --enrich` claims so they match what one narrow GraphQL page can really recover.
- Decide whether backfill should populate snapshot-oriented read models or stay limited to entity/search hydration.

### Product Hunt strategic decisions

- Keep the Atom-first baseline and GraphQL-specific historical lift as a printed-CLI policy, not a machine default.
- If Product Hunt proves a reusable "cheap incremental sync + expensive historical backfill" pattern across multiple CLIs later, revisit a machine abstraction then, with at least two non-Product-Hunt examples.

## System-Wide Impact

- **Machine default improves:** New store-backed CLIs start from a clearer freshness contract for covered command paths instead of each one inventing read-path sync wrappers.
- **Printed CLI customization remains available:** CLIs can still add API-specific enrichment or backfill layers without fighting the machine.
- **Agent behavior improves:** Hosts can call semantic read commands with fewer empty-store surprises and with better observability when JSON is enabled.
- **Review quality improves:** Generated docs become more honest about what "freshness" actually covers. Review-pipeline automation and scorecard changes stay deferred until the contract proves stable.

## Risks & Mitigations

| Risk | Severity | Why it matters | Mitigation |
|------|----------|----------------|------------|
| We overfit the machine to Product Hunt's exact architecture | High | Future CLIs inherit the wrong abstraction | Keep enrichment/backfill out of the machine scope; build only the baseline freshness contract |
| We create a second abstraction next to `spec.Cache` | High | Confusing, duplicative, harder to maintain | Extend `spec.Cache`; do not add a rival config block |
| Freshness claims still overstate coverage | High | Agents trust docs and fail in non-obvious ways | Require declared coverage and review those claims against command families |
| Explicit live-only commands still mutate local state | Medium | Behavioral regression and user surprise | Add template-level bypass tests and verify fixtures |
| Custom commands continue to bypass the machine helper | Medium | Per-CLI drift returns | Emit a small helper surface designed for custom-command adoption and document it in generated code comments |
| Scorecard rewards fixture-only behavior too early | Medium | Grades imply cross-CLI trust that has not been proven | Defer scorecard weighting until runtime verify succeeds on at least two printed CLIs |

## Exit Criteria

- The machine has one clear freshness abstraction for store-backed CLIs, centered on the existing cache/auto-refresh surface.
- Generated and explicitly registered custom commands can share a stable freshness helper and JSON metadata contract.
- README/SKILL language accurately describes freshness ownership and its limits.
- Verify covers stale-store refresh behavior and bypass semantics.
- Scorecard and review-pipeline automation are explicitly deferred until the runtime contract has adoption evidence.
- Product Hunt-specific work is clearly documented as printed-CLI follow-on, not silently absorbed into the machine.
