---
title: "fix: printing-press food52 retro fixes (issue #337)"
type: fix
status: active
date: 2026-04-27
origin: https://github.com/mvanhorn/cli-printing-press/issues/337
---

# fix: printing-press food52 retro fixes (issue #337)

## Overview

The food52 retro filed nine findings against the Printing Press machine. Three are P1 (silently breaks every Next.js-style site, silently breaks dogfood Examples coverage to 0/10, and silently spams the user's browser during verify-mock). Four are P2 (publish-validate friction, traffic-analysis schema drift, dead-helper accumulation, mock-mode harness mismatches). Two are P3 cleanups that ride along with their P1 parents.

This plan addresses all nine findings across seven implementation units. Each unit is independently mergeable; sequencing is for review ergonomics, not because units block each other except where noted.

The retro's #335 predecessor (the allrecipes retro #333) already shipped: `doctor` honors `http_transport`, `auth.type:none` correctly skips the auth subcommand, `html_extract` mode-`page` suppresses `<noscript>` subtrees, and the generator emits a `dryRunOK` helper for verify-friendly RunE patterns. Cross-checked against this branch: none of the food52 findings were addressed by that PR — the `html_extract` work was about noscript handling for `mode: page`, not adding `mode: next-data`; the `dryRunOK` work helps `--dry-run` guards but doesn't supply canonical positional values to mock-mode verify; the `auth.type:none` fix correctly didn't touch the food52 path because food52 already shipped without spurious auth scaffolding.

---

## Problem Frame

The food52 generation run did everything the Printing Press currently asks of it and still required substantial hand-work:

- **4 generated handlers replaced** because the `html_extract` template only knew `mode: page` and `mode: links`; food52 — and any other site (commonly Next.js pages-router) that embeds its data in a `<script id="__NEXT_DATA__">` block — gave the page-mode extractor only generic page metadata + nav links to work with. The frequency of this pattern across our browser-sniff backlog is uncalibrated; food52 is the first to surface it concretely. The fix is small enough to be worth shipping on N=1, but the framing in earlier drafts ("every Next.js / Nuxt / Remix site") overstated coverage — the parameterized `script_selector` (see U4) is the path to actually covering Nuxt / Remix without per-framework code paths.
- **17 source files sed'd** because the natural Go idiom for multi-line `Example:` strings (`strings.TrimSpace(\`...\`)`) silently breaks the dogfood scorer's example detection — leading 2-space indent gets stripped, dogfood's parser sees the first unindented line as a section boundary, captures nothing, scores 0/10.
- **Hand-patched `.printing-press.json`** to add `novel_features` from `research.json`, because `lock promote` writes the manifest but doesn't merge in the dogfood-verified novel_features array, so publish validate fails on every first publish.
- **`open` command emitted browser tabs to `https://food52.com/recipes/mock-value`** because verify mock-mode dispatches every command with placeholder positionals, and side-effecting commands have no convention to default-print.

Each of these is a silent failure mode — no error message points the agent at the problem until something downstream surfaces it (dogfood verdict, publish validate, the user noticing browser tabs). The fixes are individually small. The cumulative effect is meaningful: every CLI with side-effecting commands, every first-time publish, and every CLI with hand-authored Examples will behave better; future SSR-React sites that embed pageProps-shaped data in a known script tag get a path that doesn't require hand-replaced handlers.

The food52 retro is the source of truth for findings, evidence, and acceptance criteria. The retro document is reachable from issue #337 and locally at `manuscripts/food52/20260426-230853/proofs/20260427-014521-retro-food52-pp-cli.md` (relative to `~/printing-press`).

---

## Requirements Trace

- R1. F1 — Generator can extract `__NEXT_DATA__` JSON directly from a spec declaration without hand-replaced handlers (covers Next.js / Nuxt / Remix sites generically).
- R2. F2 — Dogfood detects `Example:` content regardless of indent style; generator templates emit Examples in a format that survives the natural Go idiom.
- R3. F3 — Side-effect commands (browser launch, printer dispatch, system notification) follow a print-default + explicit-opt-in convention; verify mock mode never triggers visible side effects.
- R4. F4 — `lock promote` populates the manifest's `novel_features` from the run's `research.json`, so publish validate passes on first publish without manual patching.
- R5. F5 — `traffic-analysis.json` accepts the `browser_http` reachability mode the consumer code already handles, and accepts string-shaped evidence for hand-authored discovery reports.
- R6. F6 — Generator emits `helpers.go` content selectively: helpers tied to features the spec doesn't use are not emitted.
- R7. F7 — Verify mock mode supplies canonical positional values matching `--help` placeholder names; commands requiring positionals are no longer mechanically marked failed.
- ~~R8. F8 — Generated `mode: next-data` extractor strips literal `undefined` / `null` / `NaN` tokens from extracted strings.~~ **Dropped from this plan in review round 1.** The proposed strip would mangle legitimate content (recipe instructions, programming articles, brand names). The food52 case is Sanity-CMS-specific and handled by food52's own parser. F8 stays open as a future finding if a generic pattern emerges across multiple CLIs.
- R9. F9 — SKILL template threads positional values through example invocations using the spec-default-first lookup chain (spec.Param.Default → canonicalargs → mock-value), so `verify-skill` exits 0 on first generation AND user-facing examples use real values when the spec author declares them.

**Origin actors:** Implementing agent (the next person/agent that runs `/printing-press` against any API).
**Origin flows:** F1 — `printing-press generate` → emit handlers; F2 — `printing-press dogfood` → score Examples; F3 — `printing-press verify` (mock mode) → exercise commands; F4 — `printing-press lock promote` → write manifest → publish validate; F5 — `printing-press generate --traffic-analysis` → load + validate.

---

## Scope Boundaries

- **No food52-specific changes.** None of these fixes hardcode Food52, Next.js's specific Sanity backend, Typesense, or any other API-specific detail. Every fix must work across APIs.
- **No new spec format.** The `mode: embedded-json` addition extends the existing `html_extract` block; it does not introduce a parallel "next.js spec" or replace internal YAML / OpenAPI inputs.
- **No changes to existing `syntheticArgValue` mappings.** U5 only ADDS new placeholder names via the new `internal/canonicalargs` package and reorders the lookup chain to consult the spec's `Param.Default` first. Existing per-name mappings (`slug`, `query`, `url`, `path`, `category`, `search`, `name`) stay at their current return values; changing them requires a separate audit of every library CLI's mock-mode behavior.
- **No domain-specific names in `internal/canonicalargs`.** The shared registry stays generic + cross-domain (`since`, `until`, `tag`, `vertical`). Domain-specific names like `servings`, `ingredient`, `cuisine`, `recipe_id`, `airport_code`, `team_abbr` belong in the spec author's `Param.Default`, not in the machine. AGENTS.md anti-pattern: "Never change the machine for one CLI's edge case."
- **No verify-mode redesign.** R7 fixes the placeholder-arg behavior for required positionals and the side-effect detection for visible-effect commands; it does not redesign verify's overall mock mode or migrate it to live mode.
- **No conditional-helper-emission rewrite of `helpers.go.tmpl`.** R6 (WU-6) extends the existing `HelperFlags` mechanism (which already gates `HasDelete`, `HasPathParams`, `HasMultiPositional`, `HasDataLayer`); it does not split `helpers.go.tmpl` into separate physical files unless that ends up being the simpler implementation choice during execution.
- **No backfill of existing library CLIs.** These fixes apply on the next regeneration. CLIs already in `~/printing-press/library/` are not retroactively rewritten.

### Deferred to Follow-Up Work

- **Auto-detect `__NEXT_DATA__` / `__NUXT__` / `window.__remixContext` during browser-sniff and pre-populate `html_extract.mode: embedded-json` + matching `script_selector` in the emitted spec.** WU-1 adds the mode + extractor; the auto-detection from a HAR or browser-use capture is a follow-up because it requires browser-sniff to introspect captured HTML for the Next.js script tag, which is a separate code path. Documented in the retro as part of WU-1's full goal but split out here because the extractor mode is independently shippable and useful.
- **Spec-level annotation for side-effect commands.** WU-3 uses heuristics (`--help` keywords + AST scan for `exec.Command("open"|"xdg-open"|...)`) to detect side-effecting commands. A spec-level `side_effect: true` annotation would be more declarative but requires its own design discussion and isn't needed to unblock the immediate problem.

---

## Context & Research

### Relevant Code and Patterns

- `internal/spec/spec.go` — `HTMLExtract` struct (currently `Mode` / `LinkPrefixes` / `Limit`), `HTMLExtractMode*` constants, `EffectiveMode()`. The validation at the bottom of the file enforces the mode enum. WU-1 extends this struct + enum.
- `internal/generator/templates/html_extract.go.tmpl` — runtime extractor emitted into every CLI. Currently switches on Mode (`page` / `links`). WU-1 adds the `embedded-json` branch here.
- `internal/generator/generator.go` — `HelperFlags` struct + `computeHelperFlags(spec)`. Already gates emission of `classifyDeleteError`, `replacePathParam`, `usageErr`, provenance helpers. WU-6 extends this pattern with HTML-specific flags.
- `internal/pipeline/dogfood.go` — `extractExamplesSection`. The bug is the loop body's `if len(line) > 0 && line[0] != ' ' && line[0] != '\t' { break }`, which treats any unindented line as a section boundary. WU-2 fixes this to break only on a recognized Cobra section header.
- `internal/pipeline/publish.go` — `writeCLIManifestForPublish(state, dir)`. WU-4 extends this to read `state.PipelineDir + "/research.json"` and populate `manifest.NovelFeatures` from `novel_features_built` (preferred) or `novel_features` (fallback).
- `internal/pipeline/runtime_commands.go` — `syntheticArgValue(name string) string`. Already maps `id` → `12345`, `region` → `mock-city`, `password` → `mock-secret`, default `mock-value`. WU-3 extends this map for common positional placeholder names (`slug`, `query`, `url`, etc.) and adds the side-effect-detection guard.
- `internal/cli/schema.go` — JSON Schema printer for `traffic-analysis`. Hardcodes the reachability mode enum. WU-5 fixes the enum and the EvidenceRef shape.
- `internal/browsersniff/analysis.go` — `EvidenceRef` struct (requires `EntryIndex int`). The Go consumer code at `ApplyReachabilityDefaults` already accepts `browser_http` mode. WU-5 makes the Go marshaling tolerant of either object-shaped or string-shaped evidence.
- `internal/cli/publish.go:checkPublish` — the `transcendence` check that rejects `len(manifest.NovelFeatures) == 0`. WU-4 doesn't change this check; it ensures the manifest is populated upstream so this check passes naturally.
- `internal/generator/templates/skill.md.tmpl` — `firstCommandExample .Resources` template helper produces the example string for the Agent-mode and Named-Profile sections. WU-7 extends `firstCommandExample` (or the helper that backs it) to thread canonical positional values.
- `internal/pipeline/runtime_commands.go:classifyCommandKind` — the existing classifier that names commands by intent. WU-3's side-effect detection extends this with a new "side-effect" classification.
- The food52 CLI's hand-built `internal/food52/nextdata.go`, `recipe.go`, and `cleanIngredientStrings` (in `~/printing-press/library/food52/internal/food52/util.go`) are the canonical proof-of-concept for WU-1's behavior — read them as reference but don't import.

### Institutional Learnings

- The `HelperFlags` pattern in `internal/generator/generator.go` is the existing infrastructure for conditional helper emission (R6 / WU-6). It works, has tests, and is the natural place to extend.
- The recent `auth.type:none` fix (#335) shows the pattern for conditional generator behavior driven by spec content: detect a condition, route the generator's render path. WU-1 (embedded-json) and WU-6 (per-mode helper gating) follow the same shape.
- Test fixtures for HTML extraction live under `testdata/` next to the generator tests. The food52 retro left scrubbed real HTML fixtures at `~/printing-press/library/food52/internal/food52/testdata/recipes-chicken.html` and friends — useful as a reference for what `__NEXT_DATA__`-bearing HTML actually looks like, though tests should use minimal handcrafted fixtures.

### External References

- Schema.org Recipe / Article JSON-LD shapes are stable and well-documented. Next.js's `__NEXT_DATA__` shape (`props.pageProps.<route-data>` + `buildId`) is documented in the Next.js source and is consistent across major versions.
- No external docs needed for the verify mock-mode or schema fixes — the fix is self-contained to this codebase.

---

## Key Technical Decisions

- **Use the existing `HelperFlags` mechanism for conditional helper emission (WU-6), not a file split.** The infrastructure already exists and `helpers.go.tmpl` already gates several helpers conditionally. Extending it is a smaller change than splitting `helpers.go.tmpl` into 5 physical files. If during execution it becomes clear that a file split is cleaner, the implementer can choose that path — the goal (no dead helpers when consumers don't exist) is what matters.
- **`mode: embedded-json` uses a simple dot-notation `JSONPath` field, not a real JSONPath library.** The path is always `props.pageProps.<something>` for Next.js sites, and dot-notation is enough for every realistic case. Avoids a dependency.
- **`EvidenceRef` becomes a sum type via custom JSON marshal/unmarshal, not a refactor of every consumer.** Callers reading `Evidence []EvidenceRef` continue to work; the new option is "evidence may be a string, in which case the EvidenceRef carries Reason=<string> and zero EntryIndex". Schema reflects this with `oneOf: [object, string]`.
- **`syntheticArgValue` extension lives in the existing function, not a new registry.** The function is already a name → value lookup. Extending the lookup is a smaller change than introducing a parallel type-aware system.
- **Side-effect detection uses heuristics, not a spec annotation.** Heuristics (`--help` keyword scan + AST scan for known exec.Command targets) are good enough to catch the common cases without requiring spec authors to remember an annotation. The spec annotation is the deferred follow-up for cases where heuristics are wrong.
- **WU-2 fixes both the scorer parser AND the generator template.** Either alone leaves silent footguns: a tolerant parser still lets agents write fragile examples; a fixed template still trips on hand-written commands. Doing both means the natural Go idiom Just Works AND the parser is robust to whatever style developers actually use.
- **WU-4 reads `novel_features_built` first, falls back to `novel_features`.** Dogfood writes `novel_features_built` after verifying which features were actually built. If dogfood didn't run (rare but possible — `lock promote` doesn't strictly require it), fall back to the planned list rather than emitting an empty array.

---

## Open Questions

### Resolved During Planning

- **Should `mode: embedded-json` auto-detect at browser-sniff time (e.g., scan captured HTML for known Next.js / Nuxt script tags)?** Resolved: out of scope for this plan — see Deferred to Follow-Up Work. The mode is independently shippable and useful even when an agent declares it manually.
- **Should we delete the dead helpers from existing food52 + other library CLIs?** Resolved: no. Backfill is out of scope; the fix applies on next regeneration.

### Deferred to Implementation

- **Exact set of canonical placeholder name → mock value mappings for WU-3.** The minimum set is what food52 surfaced (`slug`, `query`, `url`, `vertical`, `tag`, `ingredient`, `servings`); the right full set is what the existing library's command tree actually uses. Discoverable at implementation time by scanning every published CLI's `--help`.
- **Whether to gate the `noscript` subtree suppression already in `mode: page` (added by #335) on the same `HasHTMLExtractPage` flag in WU-6.** Likely yes for symmetry, but worth confirming during implementation that the suppression isn't useful in any non-HTML context.
- **Whether `firstCommandExample` should produce a single canonical command string or expose enough to let the template assemble it.** The current helper returns a string; threading positional values may be cleanest at the helper level rather than in the template. Decide during execution.

---

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### `mode: embedded-json` extractor flow (WU-1; WU-8 dropped)

```
HTML response body
   │
   ▼
parse HTML, find element matching spec.html_extract.script_selector
(default selector: "script#__NEXT_DATA__")
   │
   ├─ no match → return error "no script tag matching <selector> found in page"
   │
   └─ match
       │
       ▼
   json.Unmarshal script's text content into map[string]any
       │
       ├─ unmarshal error → return error
       │
       └─ ok
           │
           ▼
   walk to <spec.html_extract.json_path>
   (dot-notation; e.g. "props.pageProps.recipesByTag.results" for Next.js,
    or "data.<route>" for Nuxt; missing key → typed empty)
           │
           ▼
   re-marshal as json.RawMessage
           │
           ▼
   return as the response body
   (NO post-extraction string sanitization — see U4's plan-revision note)
```

### Side-effect command detection (WU-3)

| Signal | Source | Action |
|---|---|---|
| `--help` body contains "browser", "launch", "send", "play", "open in" (case-insensitive, word-boundary) | Help text scan via `<cli> <command> --help` | Mark as side-effecting |
| Source AST contains `exec.Command("open"\|"xdg-open"\|"start"\|"lp"\|"notify-send"\|"afplay"\|"say")` | Static scan over `internal/cli/*.go` | Mark as side-effecting |
| Side-effecting AND command supports `--dry-run` | Verify dispatch | Run with `--dry-run` |
| Side-effecting AND no `--dry-run` | Verify dispatch | Skip with WARN, not FAIL |
| Not side-effecting | Verify dispatch | Run normally |

### Examples-section parsing fix (WU-2)

Change the loop break condition in `extractExamplesSection`:

| Current | Fixed |
|---|---|
| Break on any unindented non-empty line | Break on a recognized Cobra section header: `Usage:`, `Aliases:`, `Available Commands:`, `Flags:`, `Global Flags:`, `Use "<cmd> [command] --help"...`, or two consecutive blank lines |

### Manifest novel_features hydration (WU-4)

```
lock promote(state, dir)
   │
   ▼
writeCLIManifestForPublish(state, dir)
   │
   ▼
build CLIManifest from state
   │
   ▼
NEW: read state.PipelineDir + "/research.json" if exists
   │
   ├─ has novel_features_built (non-empty) → use it
   ├─ else has novel_features (non-empty) → use it
   └─ else → leave manifest.NovelFeatures empty
   │
   ▼
project to []NovelFeatureManifest{Name, Command, Description}
   │
   ▼
WriteCLIManifest(dir, manifest)
```

---

## Implementation Units

- U1. **Examples robustness + correct generator template (WU-2 / F2)**

**Goal:** Dogfood detects `Example:` content regardless of indent style; generator templates emit examples in a format that survives the natural Go idiom.

**Requirements:** R2

**Dependencies:** None.

**Files:**
- Modify: `internal/pipeline/dogfood.go` (function `extractExamplesSection`)
- Modify: `internal/pipeline/dogfood_test.go` (add cases for `strings.TrimSpace`-style and `strings.Trim(..., "\n")`-style examples)
- Modify: `skills/printing-press/SKILL.md` Phase 3 "Agent Build Checklist" — add a note that hand-authored novel-feature commands should use `Example: strings.Trim(\`...\`, "\n")` (preserves leading indent) NOT `strings.TrimSpace(\`...\`)` (strips it). Reference the dogfood failure mode this prevents.

> **Plan-revision note (review round 1):** Originally proposed an audit-and-replace pass over `internal/generator/templates/*.tmpl` for `strings.TrimSpace(...)` in Example fields. ce-feasibility verified there are zero matches today — the food52 "17 files sed'd" was over hand-authored novel-feature commands written by the absorb agent, not generator templates. Template sweep dropped; SKILL note added so future hand-authored commands get the right pattern from the start.

**Approach:**
- Replace the loop's "break on first unindented line" with "break on first line that matches a recognized Cobra section header". Use a closed set of canonical Cobra headers ONLY: `Usage:`, `Aliases:`, `Available Commands:`, `Flags:`, `Global Flags:`, plus a literal-prefix match on `Use "` for the trailing "Use \"...\" for more information..." line Cobra emits at the bottom. Any other unindented line is treated as continuation of examples.
- **Drop the "two consecutive blank lines" fallback proposed in the original revision.** ce-adversarial flagged this as fragile — Cobra's renderer doesn't guarantee double-blank between sections, and authored Examples may legitimately contain blank lines for visual grouping. The closed section-header set + "Use \" prefix" is enough; default behavior on truly-unrecognized content is "treat as example continuation", which lose-by-default is safer than misclassify.
- Update the existing dogfood test that asserts the old break behavior.

**Patterns to follow:**
- Existing helper detection logic in `internal/pipeline/dogfood.go` for parsing `--help` output.

**Test scenarios:**
- Happy path: `--help` output with `Examples:` followed by `food52-pp-cli articles browse food` (no indent) followed by `  food52-pp-cli articles browse life --json` (indented) — dogfood detects both lines, scoring 1/1 for that command.
- Happy path: `--help` output with the canonical `  food52-pp-cli x --json\n  food52-pp-cli y --json` (both indented) — same detection.
- Edge case: `--help` output with `Examples:` followed by an empty line then `Flags:` — dogfood correctly captures empty examples and breaks at `Flags:`.
- Edge case: command with no `Example:` field at all — dogfood correctly reports missing.
- Negative: a line like `cmd-with-no-leading-space` that's NOT one of the section headers should still be captured as an example, not treated as a section boundary.

**Verification:**
- Re-running dogfood against the food52 CLI as it currently exists (with `strings.Trim(..., "\n")` examples) reports `Examples: 10/10`.
- Re-running dogfood against a hypothetical CLI using `strings.TrimSpace(...)` examples ALSO reports `Examples: 10/10`.

---

- U2. **`lock promote` populates manifest novel_features fully (WU-4 / F4)**

> **Plan-revision note (review round 1):** The original U2 description said this code didn't exist. It does — `internal/pipeline/publish.go:265-274` already calls `LoadResearch(state.PipelineDir())` and populates `m.NovelFeatures` from `research.NovelFeaturesBuilt`. This unit is now scoped to the two real gaps in that existing block.

**Goal:** `printing-press lock promote` populates `.printing-press.json`'s `novel_features` for every realistic publish path, including (a) the dogfood-skipped case where only `novel_features` (planned) exists, and (b) the minimal-state case where `lock promote` runs without a pre-existing runstate.

**Requirements:** R4

**Dependencies:** None.

**Files:**
- Modify: `internal/pipeline/publish.go:265-274` (existing block — add a fallback branch: when `research.NovelFeaturesBuilt` is nil OR points to an empty slice, also try `research.NovelFeatures`)
- Modify: `internal/pipeline/publish.go` (`writeCLIManifestForPublish`) — when `state.RunID == ""` (minimal-state path called from `lock.go:236-239`'s `NewMinimalState` fallback), discover the most recent runstate directory by globbing `<runstate-root>/runs/*/research.json` keyed on the API slug and use that path; on no match, log + skip (don't fail promote)
- Modify: `internal/pipeline/state.go` — confirm `NewMinimalState` doesn't need a `RunID` field set (today it doesn't); the discovery happens at the publish boundary, not here
- Modify: `internal/pipeline/publish_test.go` (add 4 test cases below)
- Modify: `internal/pipeline/research.go` — verify `LoadResearch` already handles missing research.json gracefully (current behavior: `os.ReadFile` returns an error, caller's `if err == nil` branch skips). Add a comment if the contract needs to be made explicit.

**Approach:**
- Read the existing block at `publish.go:265-274` first; the fallback is a one-clause addition to its `if`. Don't write a parallel read.
- For the minimal-state path: `lock.go:236-239` falls back to `NewMinimalState(cliName, dir)` when `FindStateByWorkingDir` fails. That state has empty `RunID`, so `state.PipelineDir()` returns `RunPipelineDir("")` — a path that doesn't contain a run's research.json. Detect `state.RunID == ""` inside `writeCLIManifestForPublish`, and glob the per-scope runstate directory for `runs/*/research.json` files. Pick the most recent by mtime. If the API name is in the manifest (it is — `m.APIName`), filter the glob to runs whose research.json's `api_name` field matches.
- Failure modes: missing research.json (skip silently — older runs might not have one); malformed research.json (log warning to stderr but don't fail promote); IO errors on the glob (log + skip).

**Patterns to follow:**
- Existing manifest-write code path in `internal/pipeline/publish.go`. Existing graceful-failure pattern around `writeSmitheryYAML` (logs warning, doesn't fail promote). The runstate path conventions documented in `AGENTS.md` ("`~/printing-press/.runstate/<scope>/runs/<run-id>/`").

**Test scenarios:**
- Happy path: research.json has both `novel_features` (3 entries) and `novel_features_built` (2 entries — dogfood verified the third didn't build) → manifest gets the 2-entry built list. (Already passes today — assert no regression.)
- Happy path (NEW behavior): research.json has `novel_features` only (no dogfood run yet) → manifest gets the planned list. Currently fails because `NovelFeaturesBuilt == nil` short-circuits.
- Happy path (NEW behavior, minimal-state): `lock promote` called via `NewMinimalState` (RunID empty) on a CLI whose runstate exists at `~/printing-press/.runstate/<scope>/runs/<latest>/research.json` with a populated `novel_features` → glob discovers the path, manifest gets the list.
- Edge case: research.json has neither array → manifest's `novel_features` is empty/omitted; publish validate's transcendence check still fails (correct — the CLI genuinely has no novel features).
- Error path: research.json file doesn't exist → no error, manifest's novel_features just stays empty.
- Error path: research.json exists but is malformed JSON → log warning to stderr, don't fail promote; manifest's novel_features stays empty.
- Error path (minimal-state): no runstate directories exist at all → glob returns empty, log + skip, manifest's novel_features stays empty, promote succeeds.
- Integration: full promote-then-validate cycle on a fixture pipeline state with a populated research.json → publish validate's transcendence check passes.

**Verification:**
- After `lock promote --cli <api>-pp-cli --dir <work-dir>`, the library's `.printing-press.json` contains a `novel_features` array matching the dogfood-verified list from research.json (or planned list when dogfood didn't run).
- `printing-press publish validate --dir <library-dir>` exits 0 with `transcendence: PASS` for any CLI that had novel features in its research.json — including CLIs promoted via the minimal-state path.

---

- U3. **traffic-analysis schema parity (WU-5 / F5)**

**Goal:** `traffic-analysis.json` validates with the `browser_http` mode the consumer code already accepts, and accepts hand-authored string evidence alongside HAR-derived `EvidenceRef` objects.

**Requirements:** R5

**Dependencies:** None.

**Files:**
- Modify: `internal/cli/schema.go` (the `traffic-analysis` JSON Schema printer — add `browser_http` to the reachability mode enum; change evidence array items from `{ "$ref": "#/$defs/evidence_ref" }` to `{ "oneOf": [{ "$ref": "#/$defs/evidence_ref" }, { "type": "string" }] }`)
- Modify: `internal/browsersniff/analysis.go` (add custom JSON marshaling/unmarshaling on EvidenceRef — when the JSON value is a string, populate the struct's Reason field with the string and set `EntryIndex = -1` to distinguish string-derived from real-HAR-derived entries; when it's an object, behave as today)
- Modify: `internal/browsersniff/analysis_test.go` (add roundtrip tests for both shapes; load a traffic-analysis.json with `browser_http` mode + string evidence and verify it succeeds)

**Approach:**
- The reachability mode enum addition is a one-line change. The schema currently lists `["standard_http", "browser_clearance_http", "browser_required", "blocked", "unknown"]`; add `"browser_http"`.
- For evidence as a sum type, the cleanest Go approach is implementing `UnmarshalJSON` on `EvidenceRef` that tries `json.Unmarshal` into the struct first, and on type-mismatch error falls back to unmarshaling as a string, populating `Reason`, and setting `EntryIndex = -1`. The `MarshalJSON` either emits object form (real-HAR-derived, `EntryIndex >= 0`) or string form (when `EntryIndex == -1`, emit just the `Reason` value). This makes the round-trip stable and lets downstream consumers distinguish "first HAR entry" (`EntryIndex == 0`) from "string-derived prose evidence" (`EntryIndex == -1`).
- Update the schema's `evidence_ref` definition to allow either object form (existing) or string form (new). JSON Schema `oneOf` is the right primitive.

**Patterns to follow:**
- Existing custom JSON marshaling in the codebase if any. Otherwise the Go stdlib pattern: implement `UnmarshalJSON([]byte) error` on the type.

**Test scenarios:**
- Happy path: traffic-analysis.json with `reachability.mode: browser_http` and an evidence array containing one string ("Surf cleared the challenge; plain curl returned 429") loads cleanly via `LoadTrafficAnalysis`.
- Happy path: traffic-analysis.json with `reachability.mode: browser_http` and EvidenceRef objects (the existing HAR-derived shape) still loads as before.
- Happy path: traffic-analysis.json with mixed string + object evidence loads — string entries get Reason populated, object entries get full struct.
- Edge case: `mode: invalid_made_up_mode` is still rejected by the schema.
- Roundtrip (string-derived): marshal an EvidenceRef unmarshaled from a string, get back the original string. EntryIndex == -1 distinguishes it from real-HAR-derived entries.
- Roundtrip (object-derived): marshal an EvidenceRef unmarshaled from object form, get back the same object. EntryIndex >= 0 confirms HAR-derived.

**Verification:**
- `printing-press generate --traffic-analysis <file-with-browser_http-and-string-evidence> ...` succeeds (currently fails with `cannot unmarshal string into Go struct field`).
- `printing-press schema traffic-analysis | jq '.properties.reachability.properties.mode.enum'` includes `"browser_http"`.

---

- U4. **`mode: embedded-json` HTML extractor (WU-1 / F1 only — F8 dropped from this plan)**

> **Plan-revision note (review round 1):**
> - **Mode renamed from `next-data` to `embedded-json` and parameterized.** Original name hard-baked Next.js pages-router; Next.js 14+ app router uses streamed RSC chunks (`__next_f.push`), Nuxt uses `__NUXT__`, Remix uses `window.__remixContext`, Astro has its own serialization. Generic name + `script_selector` field (e.g., `script#__NEXT_DATA__`) covers all of these patterns with one mode.
> - **F8 (cleanExtractedString helper) dropped.** Reviewers showed the proposed substring strip would mangle legitimate recipe instructions ("undefined custard"), programming articles ("returns null when..."), and brand names ("NaN cookies"). The food52 case is a Sanity-CMS-specific serialization quirk; the right fix lives in food52's parser, not the generator. F8 stays open as a future finding if a generic pattern emerges.

**Goal:** Specs declaring `html_extract: { mode: embedded-json, script_selector: "script#__NEXT_DATA__", json_path: "props.pageProps.<x>" }` (and equivalent shapes for Nuxt / Remix / Astro / future SSR-React frameworks) produce printed CLIs that extract structured data from the named script tag directly, with no hand-replaced handlers.

**Requirements:** R1 (R8 dropped — see plan-revision note)

**Dependencies:** None.

**Files:**
- Modify: `internal/spec/spec.go` (add `HTMLExtractModeEmbeddedJSON` constant; add `ScriptSelector string` field (defaults to `script#__NEXT_DATA__` for back-compat ergonomics) and `JSONPath string` field to `HTMLExtract`; extend the validation switch; update `EffectiveMode`'s comment)
- Modify: `internal/spec/spec_test.go` (validation roundtrip cases for the new mode + json_path)
- Modify: `internal/generator/templates/html_extract.go.tmpl` (add the `case "embedded-json":` branch — parse the configured `script_selector` to find the target `<script>` tag (default selector `script#__NEXT_DATA__` for the common Next.js case), extract its text content, parse JSON, walk dot-notation path, return as RawMessage. NO post-extraction string sanitization — see plan-revision note above.)
- Modify: `internal/generator/generator_test.go` (golden tests: (a) `mode: embedded-json` with `script_selector: script#__NEXT_DATA__` against a Next.js fixture; (b) `mode: embedded-json` with `script_selector: script#__NUXT__` against a Nuxt fixture, both returning the configured json_path subtree)
- Modify: `testdata/golden/fixtures/golden-api.yaml` (add a small endpoint demonstrating `mode: embedded-json` to lock the contract in)

**Approach:**
- The spec extension is small: one constant, two new fields (`ScriptSelector`, `JSONPath`), one validation case. Default `ScriptSelector` to `"script#__NEXT_DATA__"` so the common Next.js case requires only `mode: embedded-json` + `json_path: ...` in the spec.
- The runtime extractor is ~30 lines: parse HTML and find the matching script element (use the same `golang.org/x/net/html` walker the page mode already imports — selector parsing is `tag#id` or `tag` for v1; expand later if needed), `json.Unmarshal` the script's text content into `map[string]any`, a small `walkDotPath(m, "props.pageProps.foo.bar")` helper that descends and returns the target as `json.RawMessage`, then re-marshal.
- **Restructure the dispatch in `html_extract.go.tmpl` so mode-selection happens BEFORE the page-mode parse path.** Currently lines 55-103 unconditionally parse the HTML, walk the DOM, and run `looksLikeHTMLChallenge` before switching on `Mode` at the very end. For `mode: embedded-json`, the page-mode-specific work is wasted and `looksLikeHTMLChallenge` could mis-flag a Next.js page with a generic title as a challenge. New shape: switch on `opts.Mode` at the top; each mode owns its own parsing path. This naturally pairs with U6's per-mode gating — symbols only one mode uses live behind that mode's gate.
- Auto-detection during browser-sniff is deferred (see Scope Boundaries) — for now spec authors declare it manually with the right `script_selector` for their target site.

**Technical design:** *(directional, not implementation specification — the actual generator emits Go code into the printed CLI, this sketches what that emitted code should do)*

The emitted handler for a `mode: embedded-json` endpoint (using the default `script#__NEXT_DATA__` selector) should look like:

```
extractNextDataPath(html, "props.pageProps.recipesByTag.results")
  → returns json.RawMessage of the array, or error
```

with the helper roughly:

```
nextDataRe = regexp `<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`
match → parse → walk path → re-marshal → clean strings
```

**Patterns to follow:**
- The `mode: page` and `mode: links` branches in the same file. The cliutil package's existing CleanText helper.

**Test scenarios:**
- Happy path: spec with `mode: embedded-json, json_path: "props.pageProps.recipesByTag.results"` (default `script_selector: script#__NEXT_DATA__`)` against a fixture HTML page containing `__NEXT_DATA__` with that path returns the expected JSON array.
- Happy path: a different json_path on the same fixture returns the right subtree.
- Edge case: json_path target is missing from the JSON → returns null or empty (decide which during impl), with no panic.
- Edge case: json_path is empty string → returns the whole pageProps.
- Error path: HTML has no `<script id="__NEXT_DATA__">` block → returns clear error "no __NEXT_DATA__ block in page".
- Error path: `__NEXT_DATA__` block contains malformed JSON → returns parse error.
- Integration: golden generator test produces the expected handler source code for a `mode: embedded-json` endpoint.

**Verification:**
- `golden-api.yaml` extended with a `mode: embedded-json` endpoint and golden test passes.
- An ad-hoc test against the food52 chicken category page (in food52's testdata fixtures) returns the same recipe array the food52 retro's hand-built extractor returns.

---

- U5. **Side-effect command convention + verify mock-mode positional canonicals (WU-3 / F3 + F7)**

> **Plan-revision note (review round 1):** ce-scope-guardian flagged that F3 (side-effect detection) and F7 (positional canonicals) are functionally independent and could split into U5a/U5b. After the B3 + P1.e revisions (slug/query unchanged + env-var convention added), the risk asymmetry shrank. Keeping them as one unit for simplicity, with this note: **the two parts are independently mergeable.** A reviewer can accept the canonicalargs + lookup-chain work (R7) in one commit without blocking on the side-effect detector + env-var convention (R3); the dependency relationship is one-direction (U7 depends on the canonicalargs registry only, not the side-effect logic).

**Goal:** Verify mock mode never triggers visible side effects; commands with required positionals are no longer mechanically marked failed; the convention for side-effecting commands (print-by-default + explicit opt-in + `PRINTING_PRESS_VERIFY` env-var short-circuit) is documented in skill instructions and AGENTS.md.

**Requirements:** R3, R7

**Dependencies:** None. U7 depends on this unit's canonicalargs registry (R7 part); not the side-effect logic (R3 part).

**Files:**
- Create: `internal/canonicalargs/canonicalargs.go` — new shared subpackage exporting `Lookup(name string) (string, bool)` returning the canonical mock value for a positional placeholder name. Both `internal/pipeline` (verify) and `internal/generator` (SKILL template) import from here.
- Modify: `internal/pipeline/runtime_commands.go` (`syntheticArgValue`) — add a new lookup chain: (1) caller-supplied spec `Param.Default` if non-empty, (2) `canonicalargs.Lookup`, (3) the existing per-name switch (preserved for back-compat), (4) `"mock-value"` fallback
- Add to `internal/canonicalargs`: `since`, `until`, `tag`, `vertical` — generic + cross-domain placeholder names. **Do NOT add `servings`, `ingredient`, or other recipe-domain names** — those belong in the spec author's `Param.Default`, not the generic registry. AGENTS.md anti-pattern: "Never change the machine for one CLI's edge case."
- Modify: `internal/pipeline/runtime_commands.go` — wire the spec.Param.Default lookup into the call sites that pass placeholder names (the dispatch loop has the spec available; thread it through to `syntheticArgValue` if it isn't already)
- **Do NOT touch the existing `slug`/`query`/`url`/`path`/`category`/`search`/`name` mappings** in the per-name switch; they exist with calibrated return values the existing library depends on
- Modify: `internal/pipeline/runtime_commands.go` (add a side-effect classifier — function `isSideEffectCommand(cmd *discoveredCommand, sourceDir string) bool` that scans the command's `--help` output for keyword markers AND scans the printed CLI's source tree for known shell-out patterns — `exec.Command(...)` over a known set of OS binaries, `pkg/browser` calls, etc. — using `sourceDir` as the printed CLI's root, NOT the printing-press binary's own internal/cli/)
- Modify: `internal/pipeline/runtime.go` (set `PRINTING_PRESS_VERIFY=1` in every mock-mode subprocess env; in the dispatch loop, when `isSideEffectCommand` is true, route to `--dry-run` if supported, else skip-WARN with a message naming the heuristic that fired)
- Modify: `internal/pipeline/runtime_test.go` (test cases for the new placeholder mappings + side-effect classifier)
- Modify: `skills/printing-press/SKILL.md` Phase 3 — document **two** conventions for side-effect commands:
  1. Print-by-default + explicit `--launch`/`--send`/`--play` opt-in flag (the food52 `open` command pattern).
  2. **Generated side-effect commands MUST check `os.Getenv("PRINTING_PRESS_VERIFY") == "1"` before performing any visible action.** When the env var is set, the command prints what it would do and exits 0 instead of shelling out, opening a browser, sending a notification, etc. This is defense-in-depth: even if the heuristic detector in (1) misses a side-effecting command, the generated command itself short-circuits in mock-mode.
- Modify: `AGENTS.md` — add the same two-part convention as a glossary entry (short).
- Modify: `internal/generator/templates/cliutil_probe.go.tmpl` (or add a small helper to cliutil) — emit a `cliutil.IsVerifyEnv() bool` helper into every printed CLI so authors of novel side-effect commands have a one-line check (`if cliutil.IsVerifyEnv() { fmt.Println("would launch:", url); return nil }`).

**Approach:**
- The placeholder lookup is the smallest fix. Add the missing names; let `mock-value` continue as the catch-all default.
- The side-effect classifier is heuristic. The two checks complement each other: `--help` keyword scan catches commands with descriptive help text (most), AST scan catches commands whose source obviously shells out (the rest). False positives (a command whose `--help` mentions "browser" innocuously) are rare and acceptable — the cost is "skipped in mock mode", not a failure.
- The skill + AGENTS.md note is short — a paragraph naming the convention. Don't add a new phase or gate.

**Patterns to follow:**
- Existing `syntheticArgValue` map style. Existing `classifyCommandKind` switch in the same file.

**Test scenarios:**
- Happy path (NEW canonicalargs entries): `canonicalargs.Lookup("since")` returns an ISO date string (e.g., `"2026-01-01"`); `Lookup("until")` returns a later ISO date; `Lookup("tag")` returns `"mock-tag"`; `Lookup("vertical")` returns `"mock-vertical"`.
- Happy path (spec-default-first lookup): a spec param with `default: "4"` named `servings` is passed through verify; the lookup chain returns `"4"` from the spec default before hitting canonicalargs (which doesn't have `servings`) or `mock-value`.
- Happy path (canonicalargs-only): a spec param named `tag` with no `default` set falls through to canonicalargs and returns `"mock-tag"`.
- Happy path (catch-all): a spec param with no `default` and a name NOT in canonicalargs (e.g., `airport_code`) falls through to `"mock-value"`.
- Negative (regression): `syntheticArgValue("slug")` STILL returns `"general"` (existing per-name switch arm preserved). `syntheticArgValue("query")` STILL returns `"mock-query"`. `syntheticArgValue("url")` STILL returns `"/mock/path"`.
- Negative (registry hygiene): `canonicalargs.Lookup("servings")` returns `("", false)` — recipe-domain names are NOT in the generic registry. Same for `ingredient`, `recipe_id`, `cuisine`, etc. The lookup-chain fallback (spec.Param.Default → mock-value) handles these.
- Integration: re-running food52's verify mock-mode after this lands AND after food52's spec is updated to declare `default: "4"` on the `servings` param: `scale` no longer mock-fails for missing positional.
- Happy path: a command with `Use: "get <slug>"` is invoked in mock mode as `<cli> get mock-slug` and is NOT classified as failed for missing args.
- Happy path: a command whose source calls `exec.Command("open", url)` is detected as side-effecting and skip-WARN'd in mock mode.
- Happy path: a command whose `--help` mentions "browser" is detected and skip-WARN'd in mock mode.
- Happy path: a command marked side-effecting that supports `--dry-run` is dispatched with `--dry-run` rather than skip-WARN'd.
- Edge case: a command with `--help` containing "browser" innocuously (e.g., "Output is browser-friendly HTML") — false positive is acceptable; document the boundary.
- Error path: source file unreadable → AST scan returns no findings; classifier falls back to help-text scan only.
- Negative: a regular read command (no side effects) is unaffected — exercised normally with placeholder args.
- Integration: re-running verify mock-mode against a CLI with a side-effecting `open` command no longer launches the browser; the previously-failing `print` / `scale` / `which` commands now pass with canonical positionals.

**Verification:**
- `printing-press verify --dir <food52-cli> --spec <food52-spec>` does NOT launch any browser tabs even with the `open` command in mock mode.
- `printing-press verify --dir <food52-cli> --spec <food52-spec>` reports `print PASS` and `scale PASS` (currently FAIL because mock mode supplied no positional).
- `skills/printing-press/SKILL.md` Phase 3 has a paragraph about the side-effect convention.

---

- U6. **Per-mode gating inside `html_extract.go.tmpl` (WU-6 / F6)**

> **Plan-revision note (review round 1):** The original U6 said `html_extract.go.tmpl` is "emitted unconditionally today" and proposed gating its emission. Reality: `internal/generator/generator.go:991-996` already gates emission on `g.Spec.HasHTMLExtraction()`. The "30 dead helpers" list from the retro included general-purpose helpers (`printOutput`, `filterFields`, `printCSV`, `prioritizeHeaders`, etc.) that are used by ALL generated handlers — gating those would break non-HTML CLIs. The HTML-only helpers (`walkHTML`, `applyMeta`, `htmlExtractedPage`, `htmlLink`, `firstImageSrc`, `nodeTextSuppressing`) all live in `html_extract.go.tmpl`, which is already gated. The remaining gap is per-mode gating WITHIN that file: a CLI with only `mode: next-data` shouldn't emit the page/links sub-helpers.

**Goal:** When a spec uses `html_extract` exclusively in one mode, the generated `html_extract.go` doesn't emit the helpers tied to other modes.

**Requirements:** R6

**Dependencies:** U4 (the embedded-json mode must exist before per-mode gating distinguishes it from page/links).

**Files:**
- Modify: `internal/generator/generator.go` — extend `Spec.HasHTMLExtraction()` (or add a per-mode helper like `HasHTMLExtractMode(mode string) bool`) so the template can introspect which modes are in use across the spec's resources
- Modify: `internal/generator/templates/html_extract.go.tmpl` — gate the page-mode helpers (`htmlExtractedPage`, `htmlLink`, `walkHTML`, `applyMeta`, `extractHTMLLink`, `firstImageSrc`, `nodeTextSuppressing`, `looksLikeHTMLChallenge`) behind `{{- if .HasHTMLExtractMode "page" }}`. Same for any links-mode-only helpers behind `{{- if .HasHTMLExtractMode "links" }}`. The `embedded-json` branch (added by U4) is gated similarly.
- Modify: `internal/generator/generator_test.go` (golden cases: a CLI with only `mode: embedded-json` has no page-mode helpers; a CLI with mixed modes has both)

**Approach:**
- Don't touch `helpers.go.tmpl` — its helpers are general-purpose and not the source of the dead-helper count.
- Don't change the file-level gate on `html_extract.go.tmpl` — it's already correct.
- Audit `html_extract.go.tmpl` for which helpers each mode actually uses. Symbols only the page mode references: `htmlExtractedPage`, `htmlLink`, `walkHTML`, `applyMeta`, `extractHTMLLink`, `firstImageSrc`, `nodeTextSuppressing`, `looksLikeHTMLChallenge`. The challenge-detection (added in #335) is currently called inside the unconditional parse path; per-mode gating means moving the parse + challenge check into the page-mode branch and out of the dispatch prologue (which also fixes one of the U4 sub-issues — see U4's "no story for app-router" P1.d note: structural change to short-circuit before HTML parse).
- The expected savings: a `mode: embedded-json`-only CLI omits ~7 helpers from the emitted `html_extract.go` (~150 lines). Smaller than the retro's "30 dead helpers" count because most of those weren't HTML-only to begin with.

**Patterns to follow:**
- Existing conditional gates in `helpers.go.tmpl` (`{{- if .HasDelete }}`, `{{- if .HasPathParams }}`).

**Test scenarios:**
- Happy path: a spec with `mode: page` only produces an `html_extract.go` containing all current page-mode helpers (no regression).
- Happy path: a spec with `mode: embedded-json` only produces an `html_extract.go` containing only the embedded-json branch and its dependencies; `walkHTML`, `applyMeta`, `htmlExtractedPage`, etc. are absent.
- Happy path: a spec with mixed modes (`page` + `embedded-json`) emits both branches.
- Negative: a spec with no `html_extract` doesn't emit `html_extract.go` at all (existing behavior; assert no regression).
- Golden: existing golden fixtures still produce the same generated output.

**Verification:**
- A regenerated CLI with `mode: embedded-json`-only spec has fewer helpers in `html_extract.go` than today.
- Dogfood's dead-code audit on the same CLI reports fewer dead helpers (target: the page-mode-specific ones disappear). The general-purpose helpers in `helpers.go.tmpl` are out of scope and not expected to change.

---

- U7. **SKILL template threads canonical positional values into examples (WU-7 / F9)**

**Goal:** SKILL.md generated for any CLI includes example invocations with usable positional values, so `verify-skill` exits 0 on first generation. Domain-specific values (food52's `servings`, ESPN's `sport`) come from the spec's own `Param.Default`; generic ones come from the shared `canonicalargs` registry; the catch-all is `mock-value`.

**Requirements:** R9

**Dependencies:** U5 (uses the same `internal/canonicalargs` package and spec-default-first lookup chain).

**Files:**
- Modify: `internal/generator/first_command_example.go` (or wherever `firstCommandExample` is defined) — extend the helper to assemble positional values using the same lookup chain as U5: (1) spec param's `Default` field if non-empty, (2) `canonicalargs.Lookup(name)`, (3) `"mock-value"` fallback. Returns a complete invocation string instead of just a command path.
- Modify: `internal/generator/templates/skill.md.tmpl` (the `{{$agentExample := firstCommandExample .Resources}}` blocks) — once the helper returns a complete invocation, the template just substitutes it directly; no per-positional logic in the template itself.
- Modify: `internal/generator/generator_test.go` (golden case asserting a CLI with `articles get <slug>` produces SKILL examples that include a slug — `mock-value` if no spec default; the spec default if one is set; `general` from the existing per-name switch arm if the param is named `slug` and no default is set, since the lookup falls through to `syntheticArgValue` semantics for back-compat — see U5 lookup chain)

**Approach:**
- The food52 retro problem (mock-slug in a user-facing SKILL) goes away naturally: when food52's spec is updated to declare `default: "sarah-fennel-s-best-lunch-lady-brownie-recipe"` on the recipes-get slug param (a real Food52 slug), the SKILL example uses that real slug — not `mock-slug`. Spec authors get to control what users see; the generator stops baking in synthetic placeholders.
- For specs that don't declare defaults: SKILL examples use whatever the lookup chain returns. If `verify-skill` accepts these (it should — they're well-formed invocations), exit 0 on first generation.

**Patterns to follow:**
- The Quick Start section of the SKILL template, which already builds command invocations with realistic args from the research.json narrative — that's the precedent for "spec/research-derived realistic values beat synthetic placeholders". This unit extends the same principle to the Agent-mode and Named-Profile examples.

**Test scenarios:**
- Happy path: SKILL.md for a CLI with `Use: "articles get <slug>"` includes `articles get mock-slug --agent --select ...` in both the Agent-mode section (line 176-ish) and the Named Profiles section (line 228-ish).
- Happy path: SKILL.md for a CLI whose first command takes no positionals continues to render today's behavior (no placeholder injected where none is needed).
- Edge case: a command with multiple required positionals (`Use: "browse-sub <vertical> <sub>"`) gets both placeholders.
- Edge case: a command with optional positionals (`Use: "search [<query>]"`) does not get a placeholder injected for the optional.
- Integration: regenerate the food52 CLI, run `printing-press verify-skill --dir <food52>`. Exit 0 with zero positional-args findings.

**Verification:**
- Golden test for `skill.md.tmpl` updated with the new example shape.
- `verify-skill` exits 0 on a freshly regenerated CLI.

---

## System-Wide Impact

- **Interaction graph:**
  - U1 (Examples) — touches dogfood scorer; downstream affects every CLI's published Scorecard. Visible to users via the new (correct) Examples score.
  - U2 (manifest novel_features) — touches lock promote → publish validate. Visible to users via no-longer-failing first publish.
  - U3 (schema) — touches generate's traffic-analysis loader. Visible to anyone hand-authoring traffic-analysis.json.
  - U4 (embedded-json) — adds two new spec fields + new generator template branch. Backward-compatible (existing specs unaffected). Visible via specs that opt in.
  - U5 (verify safety + positionals) — touches verify mock-mode dispatch. Visible via no browser tabs + higher mock-mode pass rate.
  - U6 (helper emission) — touches generator output. Visible via cleaner generated source + better Dead Code score.
  - U7 (SKILL positionals) — touches SKILL template + verify-skill. Visible via no first-publish verify-skill failures.
- **Error propagation:**
  - U2's research.json read MUST not fail promote — the entire promote step is end-of-run and a missing/malformed research.json should log + skip, not abort.
  - U4's `embedded-json` extractor MUST surface clear errors when the configured `script_selector` matches no element OR when the matched script's text isn't valid JSON is absent or the json_path target is missing — silent empty returns would mask broken specs.
  - U5's side-effect detector MUST not block verify on a false positive — skip-WARN is the right action, not skip-FAIL.
- **State lifecycle risks:**
  - U6's conditional helper gating risks breaking existing CLIs if a helper was thought-unused but is referenced from generated code I missed. Mitigation: golden tests and a local regeneration sweep over the existing library before merge.
- **API surface parity:**
  - `traffic-analysis.json`'s schema and the consumer code both need the same set of accepted modes — U3 fixes the schema; the Go consumer already accepts `browser_http`. Confirm no other validators (CI, downstream tools) read the schema independently.
- **Integration coverage:**
  - U2 → publish validate is a multi-stage flow; an integration test for the full sequence is in the unit's test scenarios.
  - U4 + U6 interact (embedded-json mode narrows the helper consumer set); golden tests for both confirm the combined behavior.
  - U5 + U7 share the canonical positional registry; tests should confirm consistency.
- **Unchanged invariants:**
  - The spec format remains backward-compatible. Existing specs without `mode: embedded-json` work exactly as today.
  - The publish PR shape remains unchanged. The only difference is the manifest now has `novel_features` populated by default.
  - `cliutil.FanoutRun` and `cliutil.CleanText` (mentioned in AGENTS.md) are not modified by this plan.
  - The agentic SKILL review (Phase 4.8) and Phase 4.85 output review behavior are unchanged.

---

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| U6's helper gating breaks an existing CLI by hiding a helper that's actually used | Golden tests for the existing library shapes; pre-merge sweep that regenerates a cross-section of the library and runs `go build ./...` on each |
| U4's embedded-json extractor doesn't handle a Next.js shape it should (e.g., app-router RSC payloads, app router) | Document the supported shape (`__NEXT_DATA__` with `props.pageProps`); explicitly out of scope for this iteration. App router migration is a future concern |
| U5's side-effect heuristics have a high false-positive rate | Skip-WARN (not FAIL) is the correct action on positive detection — false positives only mean "didn't exercise this command in mock", which is acceptable. Live verify still runs the command |
| U2's `strings.Trim(..., "\n")` template change conflicts with another in-flight template edit | Confirm no other open PRs touch the Example fields in the templates; the change is mechanical and small enough that conflicts resolve trivially |
| U3's evidence sum-type unmarshaling subtly changes behavior for downstream consumers expecting strict object form | The new behavior is additive (objects still work as before; strings are a new accepted shape). Add a roundtrip test asserting object-shaped evidence is unchanged |

---

## Documentation / Operational Notes

- **CHANGELOG entries:** each merged unit produces an entry under `cli` scope. U2 / U6 are likely `feat(cli): ...`; the rest are `fix(cli): ...`.
- **AGENTS.md:** U5 adds a short paragraph on the side-effect-command convention. Glossary doesn't need a new term.
- **SKILL.md (printing-press):** U5 adds a one-paragraph note in Phase 3 referencing the side-effect convention. No new phase, no new gate.
- **`docs/PIPELINE.md`:** unchanged. Phase ordering and intent are unaffected.
- **Rollout:** these are all backward-compatible. Existing library CLIs continue to work as-is. New generations pick up the fixes automatically.
- **Golden harness:** several units add golden cases. After implementing each, run `scripts/golden.sh verify`; if drift is intentional, run `scripts/golden.sh update` and explain the diffs in the commit.

---

## Sources & References

- **Origin issue:** [Issue #337 — Retro: Food52 — 9 findings, 7 work units](https://github.com/mvanhorn/cli-printing-press/issues/337)
- **Origin retro doc:** Local at `~/printing-press/manuscripts/food52/20260426-230853/proofs/20260427-014521-retro-food52-pp-cli.md`. Catbox mirror: https://files.catbox.moe/uwrw1e.md
- **Predecessor plan (allrecipes retro):** [docs/plans/2026-04-26-002-feat-printing-press-p1-machine-fixes-plan.md](2026-04-26-002-feat-printing-press-p1-machine-fixes-plan.md) — shipped via PR #335; cross-checked above so this plan only proposes work that #335 didn't already cover.
- **Predecessor PR:** [PR #335 — fix(cli): printing-press P1 machine fixes (issue #333)](https://github.com/mvanhorn/cli-printing-press/pull/335)
- **Food52 publish PR (provides the regenerate target):** [PR #137 — feat(food52): add food52](https://github.com/mvanhorn/printing-press-library/pull/137)
- **Food52 CLI source (reference for the hand-built embedded-json extractor + cleanIngredientStrings):** `~/printing-press/library/food52/internal/food52/`
- **Generator templates:** `internal/generator/templates/`
- **Spec types:** `internal/spec/spec.go`
- **Pipeline (publish/dogfood/runtime):** `internal/pipeline/`
- **Browser-sniff schema printer:** `internal/cli/schema.go`
- **Browser-sniff types:** `internal/browsersniff/analysis.go`
