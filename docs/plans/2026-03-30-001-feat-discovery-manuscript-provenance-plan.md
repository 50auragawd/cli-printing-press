---
title: "feat: Add discovery/ manuscript directory for sniff provenance"
type: feat
status: completed
date: 2026-03-30
origin: docs/brainstorms/2026-03-30-sniff-manuscript-provenance-requirements.md
---

# feat: Add discovery/ manuscript directory for sniff provenance

## Overview

Add a `discovery/` directory to the manuscript archive structure so sniff-derived CLIs preserve the raw evidence of how their API was discovered. Currently, sniff intermediate artifacts (HAR captures, URL lists) are written to the runstate root and silently dropped during archival. This adds the directory, redirects sniff artifacts into it, generates a human-readable sniff report, and updates the Go archive step to copy it.

## Problem Frame

When a CLI is generated via browser sniff, the manuscripts preserve the research brief and shipcheck but not the discovery evidence — what pages were browsed, what traffic was captured, what endpoints were derived. A future maintainer cannot reproduce or extend the sniff. The root cause: sniff artifacts live at `$API_RUN_DIR/` root, but archival only copies `research/` and `proofs/`. (see origin: docs/brainstorms/2026-03-30-sniff-manuscript-provenance-requirements.md)

## Requirements Trace

- R1. Discovery artifacts go to `$API_RUN_DIR/discovery/`, peer to research/ and proofs/
- R2. Archive step copies discovery/ to manuscripts
- R3. publish package includes discovery/ (already works — CopyDir copies the full tree)
- R4. Skill generates sniff-report.md in discovery/
- R5. Sniff report includes content-aware truncated response samples
- R6. HAR archived with response bodies stripped
- R7. Deduplicated URL/path list archived in discovery/

## Scope Boundaries

- Phase 1 only (sniff provenance). Crowd-sniff (R8-R9) deferred to Phase 2 after PR #67 merges
- No binary `--json` provenance output changes — skill-generated reports only
- No changes to publish validate manuscripts check (stays warn-only)
- No retroactive provenance for existing CLIs

## Context & Research

### Relevant Code and Patterns

- `internal/pipeline/paths.go` — All path functions (RunResearchDir, ArchivedResearchDir, etc.). New discovery paths follow this exact pattern.
- `internal/pipeline/state.go:561-566` — PipelineState methods (ResearchDir, ProofsDir). New DiscoveryDir follows the one-liner pattern.
- `internal/pipeline/publish.go:122-162` — ArchiveRunArtifacts iterates a hardcoded `pairs` slice. Missing dirs are silently skipped via `os.IsNotExist -> continue`.
- `internal/pipeline/paths_test.go:21-27` — Path assertions. New paths get the same test pattern.
- `internal/cli/publish.go:234-248` — publish package uses `CopyDir` on the entire `manuscripts/<api>/<runID>/` tree. No changes needed — discovery/ is automatically included.
- `skills/printing-press/SKILL.md:981-989` — Shell-based archive step copies research/ and proofs/ only. Needs discovery/ line added.
- `skills/printing-press/SKILL.md:507-635` — Sniff flow writes `sniff-urls.txt`, `sniff-unique-paths.txt`, `sniff-capture.har/.json` to `$API_RUN_DIR/` root.

### Institutional Learnings

- `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md` — Authoritative layout contract. Active runs in `.runstate/`, archived in `manuscripts/<api>/<runID>/`. Path logic centralized in `internal/pipeline/paths.go`.
- `docs/solutions/best-practices/validation-must-not-mutate-source-directory-2026-03-29.md` — Publish validation must not mutate source. Not directly triggered here since we're adding to the archive side, not validation.

## Key Technical Decisions

- **No publish.go changes needed**: Research confirmed `publish package` copies the entire `manuscripts/<api>/<runID>/` tree via `CopyDir`. Once `ArchiveRunArtifacts` creates `discovery/`, it flows through automatically. (see origin: Outstanding Questions on R3, now resolved)
- **Go-side + skill-side archive**: The Go `ArchiveRunArtifacts` handles the fullrun pipeline path. The skill's shell archive handles the skill-driven path. Both need the discovery/ pair.
- **HAR body stripping in the skill**: The skill orchestrates the sniff and has the HAR file. A `jq` command in the skill strips response bodies before archiving. No binary utility needed for Phase 1.
- **Sniff report is skill-generated markdown**: The skill has full context (pages visited, endpoints found, rate events, proxy detection). It writes the report directly — no binary `--json` intermediate needed.

## Open Questions

### Resolved During Planning

- **Does publish package need changes for discovery/?** No. CopyDir copies the full tree. Confirmed by reading `internal/cli/publish.go:234-248`.
- **Report naming convention?** Use `sniff-report.md` (no timestamp). Matches `brief.md` / `shipcheck.md` pattern — one per run, overwritten on re-run.

### Deferred to Implementation

- Exact `jq` expression for stripping HAR response bodies — depends on HAR structure from the specific sniff backend
- Whether `sniff-urls.txt` (raw) is worth archiving alongside `sniff-unique-paths.txt` (deduped), or if only the deduped list adds value

## Implementation Units

- [ ] **Unit 1: Add discovery/ path functions and archive pair**

  **Goal:** Establish the discovery/ directory in the pipeline's path system and wire it into ArchiveRunArtifacts.

  **Requirements:** R1, R2

  **Dependencies:** None

  **Files:**
  - Modify: `internal/pipeline/paths.go`
  - Modify: `internal/pipeline/state.go`
  - Modify: `internal/pipeline/publish.go`
  - Test: `internal/pipeline/paths_test.go`

  **Approach:**
  - Add `RunDiscoveryDir(runID string) string` and `ArchivedDiscoveryDir(apiName, runID string) string` to paths.go, following the RunResearchDir/ArchivedResearchDir pattern
  - Add `func (s *PipelineState) DiscoveryDir() string` to state.go, following the ResearchDir one-liner pattern
  - Add the `{src: state.DiscoveryDir(), dst: ArchivedDiscoveryDir(...)}` pair to the `pairs` slice in ArchiveRunArtifacts (publish.go:133-137)

  **Patterns to follow:**
  - `RunResearchDir` / `ArchivedResearchDir` in paths.go (lines 84-118)
  - `func (s *PipelineState) ResearchDir()` in state.go (line 561)
  - The `pairs` slice in ArchiveRunArtifacts (publish.go:133-137)

  **Test scenarios:**
  - Happy path: `RunDiscoveryDir("run-123")` returns `<runRoot>/run-123/discovery`
  - Happy path: `ArchivedDiscoveryDir("notion", "run-123")` returns `<manuscripts>/notion/run-123/discovery`
  - Happy path: ArchiveRunArtifacts copies discovery/ when it exists
  - Edge case: ArchiveRunArtifacts silently skips discovery/ when it does not exist (existing behavior for missing dirs)
  - Integration: Full archive cycle — create discovery/ with a test file, run ArchiveRunArtifacts, verify file appears in archived location

  **Verification:**
  - `go test ./internal/pipeline/...` passes
  - New path functions return correct paths under test PRESS_HOME
  - ArchiveRunArtifacts copies discovery/ alongside research/ and proofs/

- [ ] **Unit 2: Redirect sniff artifacts to discovery/ in the skill**

  **Goal:** Update the skill's sniff flow to write all intermediate artifacts into `$API_RUN_DIR/discovery/` instead of the run root.

  **Requirements:** R1, R6, R7

  **Dependencies:** Unit 1 (discovery/ must be a recognized directory)

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - Add `DISCOVERY_DIR="$API_RUN_DIR/discovery"` to the run init block (near line 163, alongside RESEARCH_DIR and PROOFS_DIR)
  - Add `mkdir -p "$DISCOVERY_DIR"` to the directory creation (near line 165)
  - Update Step 2a.2 to write `sniff-urls.txt` to `$DISCOVERY_DIR/sniff-urls.txt` instead of `$API_RUN_DIR/sniff-urls.txt`
  - Update Step 2a.3 to write `sniff-unique-paths.txt` to `$DISCOVERY_DIR/sniff-unique-paths.txt`
  - Update Steps 2a.4 / 2b to write `sniff-capture.json` / `sniff-capture.har` to `$DISCOVERY_DIR/`
  - Update the archive step (lines 986-988) to add: `cp -r "$DISCOVERY_DIR" "$PRESS_MANUSCRIPTS/<api>/$RUN_ID/discovery" 2>/dev/null || true`
  - Add a HAR body-stripping step before archival: use `jq` to remove `.log.entries[].response.content.text` from the HAR before copying to discovery/

  **Patterns to follow:**
  - Existing `$RESEARCH_DIR` / `$PROOFS_DIR` variable definitions and mkdir pattern in SKILL.md
  - Existing archive step pattern (cp -r with 2>/dev/null || true)

  **Test scenarios:**
  - Not directly testable in Go (skill is markdown). Verified by running a sniff-based generation and confirming discovery/ contains the expected files.

  **Verification:**
  - After a sniff run, `$API_RUN_DIR/discovery/` contains: `sniff-urls.txt`, `sniff-unique-paths.txt`, `sniff-capture.har` (bodies stripped) or `sniff-capture.json`
  - After archive, `$PRESS_MANUSCRIPTS/<api>/<run-id>/discovery/` contains the same files
  - No sniff artifacts remain at the `$API_RUN_DIR/` root level

- [ ] **Unit 3: Generate sniff-report.md in the skill**

  **Goal:** At the end of the sniff flow, the skill produces a structured sniff report summarizing the discovery evidence.

  **Requirements:** R4, R5

  **Dependencies:** Unit 2 (artifacts must be in discovery/)

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - After Step 4 ("Report and update spec source") and before the generate phase, add a "Write sniff discovery report" step
  - The skill writes `$DISCOVERY_DIR/sniff-report.md` containing:
    - **Pages Visited** — list of URLs browsed, in order
    - **Sniff Configuration** — backend used, pacing settings, proxy pattern detection result
    - **Endpoints Discovered** — table of method, path, status code, content type
    - **Coverage Analysis** — what resource types were exercised, what was likely missed based on research brief comparison
    - **Response Samples** — for each unique response shape, first 2KB or 100 lines (whichever smaller) of JSON/text content. Binary responses get a metadata note (content type, size) instead
    - **Rate Limiting Events** — any 429s encountered, effective rate achieved
  - Content-aware truncation: the skill checks content type before including response samples. JSON/text → truncate. Binary → metadata note only.

  **Patterns to follow:**
  - The skill's existing research brief generation (Phase 1 brief structure)
  - The skill's existing shipcheck report generation (proofs structure)

  **Test scenarios:**
  - Not directly testable in Go (skill-generated markdown). Verified by running a sniff-based generation and confirming sniff-report.md exists with expected sections.

  **Verification:**
  - After a sniff run, `$DISCOVERY_DIR/sniff-report.md` exists
  - Report contains all 6 required sections (pages, config, endpoints, coverage, samples, rate events)
  - JSON response samples are truncated to 2KB max
  - Binary response types show metadata note instead of content
  - Report is archived to manuscripts alongside other discovery files

- [ ] **Unit 4: Update layout contract documentation**

  **Goal:** Update the authoritative output layout documentation to include discovery/ as a recognized manuscript subdirectory.

  **Requirements:** Keeps docs/solutions/ accurate

  **Dependencies:** Units 1-3

  **Files:**
  - Modify: `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md`

  **Approach:**
  - Add `discovery/` to the manuscripts directory listing alongside research/, proofs/, pipeline/
  - Note that discovery/ is optional (only present for sniff/crowd-sniff derived CLIs)
  - Keep the update minimal — just the directory listing and a one-line description

  **Patterns to follow:**
  - Existing directory listing format in the layout contract doc

  **Test scenarios:**
  - N/A (documentation)

  **Verification:**
  - Layout contract mentions discovery/ as an optional manuscript subdirectory

## System-Wide Impact

- **Interaction graph:** ArchiveRunArtifacts is called from fullrun.go after PublishWorkingCLI. The skill's shell archive step runs independently. Both paths now copy discovery/. publish package downstream copies the full tree — no interaction change.
- **Error propagation:** Missing discovery/ is silently skipped (same as missing research/ or proofs/). No new error paths introduced.
- **State lifecycle risks:** When sniff flow is re-executed in an existing `$API_RUN_DIR`, prior `discovery/` contents are overwritten. This is expected — each sniff run produces fresh artifacts. Discovery/ is write-once per sniff execution, archived once, then read-only.
- **API surface parity:** No CLI flags, commands, or public APIs change. Only internal directory structure.
- **Unchanged invariants:** publish validate manuscripts check stays warn-only. Existing CLIs without discovery/ are unaffected. The binary `sniff` command output is unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Skill changes are not directly testable in Go | Manual verification via a sniff run. Unit 1 (Go changes) is fully testable. |
| HAR body stripping via jq may vary by HAR format | The `jq` expression targets the standard HAR 1.2 `.log.entries[].response.content.text` path. Both browser-use and agent-browser produce standard HAR. |
| Large HAR files even after body stripping | Headers-only HARs are typically <1MB for a 10-15 page sniff session. Acceptable for manuscripts. |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-30-sniff-manuscript-provenance-requirements.md](docs/brainstorms/2026-03-30-sniff-manuscript-provenance-requirements.md)
- Related code: `internal/pipeline/paths.go`, `internal/pipeline/publish.go`, `internal/pipeline/state.go`
- Layout contract: `docs/solutions/best-practices/checkout-scoped-printing-press-output-layout-2026-03-28.md`
- Skill definition: `skills/printing-press/SKILL.md`
