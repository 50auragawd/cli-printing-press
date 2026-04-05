---
title: "feat: Deep verification for printed CLIs"
type: feat
status: completed
date: 2026-04-04
---

# feat: Deep verification for printed CLIs

## Overview

Formalize the skill's dogfood testing as a non-skippable acceptance gate with structured pass/fail criteria, and extend the data pipeline test to validate that sync actually writes rows. These are the two remaining verification gaps — HTTP error detection is already handled by `classifyAPIError()` in the generated templates.

## Problem Frame

The skill's Phase 5 dogfood testing catches real problems (broken auth, missing data, bad response quality) but it's optional — the user can select "Skip testing." When skipped, the CLI ships with only mechanical verification (exit codes, dry-run, help text). Exit codes catch HTTP errors (401/403/404/5xx are already mapped to non-zero exit codes by `classifyAPIError()` in `helpers.go.tmpl`), but they can't validate:

- **Response quality**: Does the data look correct? Does a list return actual items?
- **Data pipeline integrity**: Did sync write rows to SQLite? Does search find results?
- **End-to-end workflows**: Does sync → sql → search → health chain actually produce meaningful output?

These require agent judgment (picking the right commands, understanding the API's data) — not mechanical parsing.

### Why not stdout parsing?

During planning, document review revealed that the generated CLI templates already have comprehensive HTTP error handling:

- `classifyAPIError()` in `helpers.go.tmpl` maps HTTP 401→exit 4, 403→exit 4, 404→exit 3, 429→exit 7, 5xx→exit 5
- `main.go.tmpl` calls `os.Exit(cli.ExitCode(err))` on any error
- `sanitizeJSONResponse()` strips JSONP/XSSI prefixes (the root cause of the Redfin false-pass scenario)

Verify's exit-code checking already catches these cases. Building a parallel stdout-parsing layer (regex-matching for error codes in CLI output) would be fragile, coupled to template versions, and redundant with the structured exit code mechanism the machine already controls.

### Evidence from retros

- **Redfin (52% verify)**: The false-pass was caused by JSONP prefix `{}&&` corrupting JSON parsing *before* the status code check ran. This has been fixed — `sanitizeJSONResponse()` now strips these prefixes.
- **Cal.com (89% verify)**: Missing `cal-api-version` header caused 404s. Fixed by required header detection (#125). With the header present, `classifyAPIError()` correctly maps 404→exit 3.
- **Steam (75% verify)**: False negatives from verify not passing auth env vars. Fixed in verify harness, not a template issue.

### What's still missing

The data pipeline test (`runDataPipelineTest()` in `runtime.go:570-597`) only checks "sync doesn't crash" — it doesn't validate that tables were created or rows were written. A sync that exits 0 with 0 rows is a silent failure that exit codes can't detect.

## Requirements Trace

### Dogfood Gate (Skill)

- R1. The skill's Phase 5 dogfood testing becomes a formal gate — not skippable when an API key is available
- R2. Dogfood produces a structured, machine-readable report that can be archived and compared across runs
- R3. When the dogfood gate fails and the agent exhausts fix-loop attempts, the CLI goes on hold (connects to existing hold mechanism)

### Data Pipeline Validation (Verify Command)

- R4. After sync, verify checks that domain tables were created in SQLite
- R5. In live mode, verify checks that sync wrote >0 rows to at least one domain table
- R6. Data pipeline result reported as PASS/WARN/SKIP with detail, not just boolean

## Scope Boundaries

- **In scope**: Non-skippable dogfood gate, structured acceptance report, deeper data pipeline validation in verify
- **Not in scope**: Stdout parsing / response body analysis in verify — `classifyAPIError()` already handles HTTP error detection via exit codes
- **Not in scope**: OpenAPI schema validation of response fields
- **Not in scope**: Mock server intelligence
- **Not in scope**: Mutation testing in verify — mutations stay in the agent-driven dogfood
- **Dropped from original plan**: Units 1-3 (runCLIWithOutput, analyzeOutput, response quality reporting) — invalidated by discovery that `classifyAPIError()` + `sanitizeJSONResponse()` already produce structured non-zero exit codes for HTTP errors

## Context & Research

### Relevant Code and Patterns

- `internal/pipeline/runtime.go:570-597` — `runDataPipelineTest()`: tests sync→health chain but only checks "sync doesn't crash," not "sync wrote rows"
- `internal/pipeline/runtime.go:600-611` — `runCLI()`: returns only error. `runCLIWithOutput()` variant needed for Unit 2 to parse sql command output
- `internal/generator/templates/helpers.go.tmpl:141-237` — `classifyAPIError()`: comprehensive HTTP error→exit code mapping. Covers 401, 403, 404, 409, 429, 5xx.
- `internal/generator/templates/client.go.tmpl:362` — `sanitizeJSONResponse()`: strips JSONP/XSSI prefixes before parsing (fixes the Redfin root cause)
- `internal/generator/templates/main.go.tmpl:14-16` — `os.Exit(cli.ExitCode(err))`: ensures non-zero exit on any classified error
- `skills/printing-press/SKILL.md:1297-1367` — Phase 5 dogfood protocol: structured test list with quick check (6 tests) and full dogfood (13 tests). Currently optional despite being marked "MANDATORY."

### Institutional Learnings

- **Redfin retro finding #4**: JSONP prefix caused JSON parse failure before status check. Root cause fixed by `sanitizeJSONResponse()`. Exit codes now work correctly for this class of bug.
- **Cal.com retro finding #5**: Missing required headers fixed by header detection (#125). With headers present, `classifyAPIError()` catches 404s.
- **Prior plans**: Two prior verification plans (2026-03-26 proof-of-behavior, 2026-03-27 runtime verification) created the current verify command. This plan narrows to the remaining gaps rather than adding another parsing layer.

## Key Technical Decisions

- **Don't parse stdout for error codes**: The generated CLI templates already map HTTP errors to structured exit codes. Building a parallel regex-based detection path would be fragile, coupled to template versions, and redundant. Trust the machine's own error handling.
- **Dogfood gate is skill-enforced, not binary-enforced**: The skill already has the Phase 5 protocol. Making it non-skippable is a skill text change. The binary doesn't know if an API key was available during generation.
- **Connect to existing hold mechanism**: When the dogfood gate fails and fix loops are exhausted, the CLI goes on hold via the existing `printing-press lock` mechanism (SKILL.md:1293-1295), not a new enforcement path.
- **`runCLIWithOutput()` for data pipeline only**: A new function that returns stdout for parsing, but scoped to Unit 2's data pipeline validation (parsing `sql` command output), not for general response analysis.

## Open Questions

### Resolved During Planning

- **Q: Should verify parse stdout for HTTP error codes?** No. `classifyAPIError()` in the templates already maps HTTP errors to non-zero exit codes. The Redfin false-pass was caused by JSONP prefix corruption, now fixed by `sanitizeJSONResponse()`. Stdout parsing would be redundant.
- **Q: Should Layer 2 be a new printing-press subcommand or stay in the skill?** Stay in the skill. The value is agent judgment — picking which commands to test deeply based on the API.
- **Q: What happens when dogfood gate fails and fix loops are exhausted?** CLI goes on hold via existing mechanism. User can retry or release the lock.

### Deferred to Implementation

- **Q: How should verify discover domain table names for row-count validation?** Options: parse `sqlite_master`, use the `health` command output, or query via `sql "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite%' AND name NOT LIKE '%_fts%'"`. Implementation should test which approach is most reliable across CLIs.

## Implementation Units

- [ ] **Unit 1: Formalize dogfood gate in skill**

**Goal:** Make Phase 5 dogfood testing non-skippable when an API key is available, require a structured acceptance report, and connect to the existing hold mechanism for gate failures.

**Requirements:** R1, R2, R3

**Dependencies:** None

**Files:**
- Modify: `skills/printing-press/SKILL.md`
- Modify: `skills/printing-press/references/dogfood-testing.md`

**Approach:**
- SKILL.md:1299 already declares Phase 5 "MANDATORY when an API key is available." The actual change is removing the contradictory option 3 ("Skip testing") at line 1312 and adding the structured report format.
- Replace the three-option prompt with: "Quick check (recommended)" / "Full dogfood" — no skip option. When no API key is available, skip is automatic and documented.
- Add a structured report format requirement:
  ```
  Acceptance Report: <cli-name>
  Level: Quick Check / Full Dogfood
  Tests: N/M passed
  Failures: [list with command, expected, actual]
  Gate: PASS / FAIL
  ```
- Define acceptance threshold: Quick Check must pass 5/6 core tests. Full Dogfood must pass 10/13. Any auth or sync failure is an automatic gate FAIL.
- The report artifact goes to `$PROOFS_DIR/<stamp>-fix-<api>-pp-cli-acceptance.md`
- Add gate failure path: when dogfood gate=FAIL and agent exhausts fix loops (max 2), the CLI goes on hold via existing `printing-press lock hold` mechanism. Document that the user can retry or release.
- Phase 5.6 (Promote) checks for the acceptance artifact — if it doesn't exist and an API key was available, promotion is blocked.

**Patterns to follow:**
- `skills/printing-press/SKILL.md:1297-1367` — existing Phase 5 structure
- `skills/printing-press/SKILL.md:1293-1295` — existing hold/lock mechanism

**Test scenarios:**
- Happy path: API key available → user sees "Quick check (recommended)" / "Full dogfood" with no skip option → tests run → acceptance report written
- Edge case: no API key available → Phase 5 auto-skips with documented reason → no acceptance artifact → promotion still allowed
- Error path: Quick Check fails 2/6 tests → gate=FAIL → agent fixes and retries → still fails → CLI goes on hold
- Integration: Phase 5.6 checks for acceptance artifact presence when API key was known to be available

**Verification:**
- Reading the modified SKILL.md shows no skip option when API key is available
- The acceptance report format is clearly specified with machine-parseable structure
- Gate failure path connects to existing hold mechanism

---

- [ ] **Unit 2: Data pipeline deep test in verify**

**Goal:** Extend `runDataPipelineTest()` to validate that sync actually creates tables and writes rows — not just "sync doesn't crash."

**Requirements:** R4, R5, R6

**Dependencies:** None

**Files:**
- Modify: `internal/pipeline/runtime.go` (the `runDataPipelineTest` function)
- Test: `internal/pipeline/runtime_test.go`

**Approach:**
- Add `runCLIWithOutput()` — a variant of `runCLI()` that returns `(stdout []byte, error)` using `cmd.CombinedOutput()` (no need to separate stdout/stderr for this use case — we just need to parse the sql command's text output)
- After sync completes, run `sql "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite%' AND name NOT LIKE '%_fts%' AND name != 'sync_state'"` to discover domain tables
- Parse the output to get table names. If 0 domain tables exist, DataPipeline=WARN
- In live mode with API key: run `sql "SELECT count(*) FROM <first-domain-table>"` for the first discovered table. If count is 0 after a full sync against a live API, DataPipeline=WARN
- In mock mode: skip the row-count check (mock data is minimal, 0 rows expected)
- Run `health` and check for non-empty output (exit 0 + non-empty = health works)
- Report data pipeline result as `PASS` (tables created + rows written in live), `WARN` (tables created but 0 rows in live), `SKIP` (mock mode or no sync command), or `FAIL` (sync crashes — existing behavior preserved)
- Update `DataPipeline` field in `VerifyReport` from `bool` to a string with the PASS/WARN/SKIP/FAIL detail

**Patterns to follow:**
- `internal/pipeline/runtime.go:570-597` — existing `runDataPipelineTest()` structure
- `internal/pipeline/runtime.go:600-611` — existing `runCLI()` as base for `runCLIWithOutput()`

**Test scenarios:**
- Happy path (live mode): sync writes rows → sql count > 0 → DataPipeline=PASS
- Error path (live mode): sync exits 0 but writes 0 rows → DataPipeline=WARN
- Edge case (mock mode): sync exits 0, tables created but 0 rows → DataPipeline=PASS (expected in mock)
- Edge case: CLI has no sync command → DataPipeline=SKIP
- Edge case: sql command not available or fails → fall back to existing boolean check (sync doesn't crash = PASS)
- Edge case: domain table names contain SQL reserved words (quoted identifiers) → query still works
- Error path: sync crashes → DataPipeline=FAIL (existing behavior preserved)

**Verification:**
- `go test ./internal/pipeline/...` passes
- Running verify on cal-com-pp-cli in live mode shows DataPipeline=PASS with table names and row count
- Running verify in mock mode shows DataPipeline=PASS (no row-count check)
- `VerifyReport.DataPipeline` field change is backward-compatible (consumers checking truthiness still work if the string is non-empty for PASS)

## System-Wide Impact

- **Interaction graph:** Unit 1 changes skill text only — no binary consumers affected. Unit 2 changes `VerifyReport.DataPipeline` from `bool` to `string` — the scorecard reads this field. The string values are truthy/falsy compatible ("PASS"/"WARN" are truthy, "" is falsy), so existing boolean consumers survive.
- **Error propagation:** The dogfood gate adds a new blocking path — if dogfood fails and fix loops are exhausted, the CLI goes on hold. This is intentional and connects to the existing hold mechanism.
- **State lifecycle risks:** None for Unit 2 (verify is stateless). Unit 1's acceptance artifact is a new file in the manuscripts directory — follows existing proof artifact patterns.
- **Unchanged invariants:** Verify's per-command pass/fail scoring is unchanged. `classifyAPIError()` exit codes continue to work as before. The `VerifyReport` struct's `Total`, `Passed`, `Failed`, `PassRate`, `Verdict` fields keep their current semantics.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Dogfood gate blocks shipping on API-specific issues (rate limits, permissions) | Quick Check tests are read-only and lightweight. Agent distinguishes "API limitation" vs "CLI bug" in the report. Exhausted fix loops → hold, not permanent block. |
| `sql` command interface varies across CLIs (flags, positional args) | Use the simplest invocation: `sql "SELECT ..."` as a positional arg. Fall back to existing boolean check if sql fails. |
| `DataPipeline` field type change (bool → string) breaks consumers | String values are truthy/falsy compatible. Add `DataPipelineDetail` as a new field if backward compatibility requires keeping `DataPipeline` as bool. |
| Domain table discovery query may miss tables with unusual naming | The `sqlite_master` query excludes only known system tables (sqlite%, %_fts%, sync_state). Any other table is treated as a domain table. |

## Sources & References

- Redfin retro: `docs/retros/2026-03-30-redfin-retro.md` — finding #4 (JSONP prefix root cause, now fixed)
- Cal.com retro: `docs/retros/2026-04-04-cal-com-retro.md` — finding #5 (required headers, now fixed)
- Key code: `internal/pipeline/runtime.go:570-597` (`runDataPipelineTest`), `internal/generator/templates/helpers.go.tmpl:141-237` (`classifyAPIError`)
- Existing mechanism: `skills/printing-press/SKILL.md:1293-1295` (hold/lock for gate failures)
- Document review: 5-persona review on 2026-04-04 revealed `classifyAPIError()` invalidates stdout parsing approach, leading to this narrowed scope
