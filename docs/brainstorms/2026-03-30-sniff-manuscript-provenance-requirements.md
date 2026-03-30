---
date: 2026-03-30
topic: discovery-manuscript-provenance
---

# Discovery Manuscript Provenance

## Problem Frame

When a CLI is generated via a discovery method (browser sniff, HAR import, or the upcoming crowd-sniff from PR #67), the manuscript artifacts fail to preserve the discovery evidence. The research brief captures product context and the shipcheck captures quality gates, but nothing records *how* the API was discovered: what was observed, what was inferred, and what confidence backs each endpoint.

This matters for two reasons:
- **Reproducibility**: A future maintainer extending the CLI cannot tell what was discovered, what was missed, or how to re-run the process.
- **Auditability**: A reviewer cannot verify that the generated spec reflects real API behavior rather than hallucinated endpoints.

The root cause is structural: discovery flows write intermediate artifacts directly into `$API_RUN_DIR/`, but the archive step only copies `research/` and `proofs/`. Discovery artifacts are silently dropped.

```
$API_RUN_DIR/
  research/               <-- archived to manuscripts
    brief.md
    spec.yaml
  proofs/                 <-- archived to manuscripts
    shipcheck.md
  sniff-urls.txt          <-- LOST (browser sniff)
  sniff-capture.har       <-- LOST (browser sniff)
  sniff-unique-paths.txt  <-- LOST (browser sniff)
  state.json              <-- LOST
```

There are currently two discovery methods with different provenance needs. Browser sniff is shipped (Phase 1.7 in the skill). Crowd-sniff is pending (PR #67, not yet merged).

| Method | Status | Source | Key evidence |
|--------|--------|--------|-------------|
| **Browser sniff** | Shipped | One browsing session on the live site | Pages visited, HAR/capture, response samples |
| **Crowd-sniff** | Pending PR #67 | npm SDKs + GitHub code search | SDKs analyzed, repos searched, source tiers, frequency counts |

Both produce spec YAML that gets archived in `research/`, but the raw evidence that produced each spec is lost.

## Implementation Phases

This work is phased to avoid coupling to the unmerged crowd-sniff PR.

- **Phase 1** (this plan): Sniff provenance — discovery/ directory, archive step, sniff report, raw evidence archival. Lands independently.
- **Phase 2** (after PR #67 merges): Crowd-sniff provenance — crowd-sniff report, crowd-sniff evidence archival. Reuses the discovery/ infrastructure from Phase 1.

The discovery/ directory structure supports both methods from day one. Phase 2 adds content without restructuring.

## Requirements

**Discovery Directory and Archival**

- R1. Each discovery method must write its artifacts into a dedicated `$API_RUN_DIR/discovery/` directory, peer to `research/` and `proofs/`. Within `discovery/`, artifacts are prefixed by method (e.g., `discovery/sniff-*`, `discovery/crowd-*`). If no discovery method ran, the directory is absent — archival and packaging skip it gracefully.
- R2. The skill's archive step must copy `discovery/` to `$PRESS_MANUSCRIPTS/<api>/<run-id>/discovery/` alongside `research/` and `proofs/`.
- R3. The `publish package` command must include `.manuscripts/<run-id>/discovery/` when manuscripts are present and a discovery directory exists. Discovery/ is optional — its absence is normal for non-discovery CLIs and pre-discovery runs, not an error.

**Sniff Discovery Report (Skill-Generated) — Phase 1**

- R4. At the end of the browser sniff flow, the skill must produce a structured sniff report at `$API_RUN_DIR/discovery/sniff-report.md` containing:
  - Pages visited (URLs and order)
  - Endpoints discovered (method, path, status code observed)
  - Coverage analysis: what resource types were exercised, what was likely missed
  - Proxy pattern detection result (if applicable)
  - Effective sniff rate and any rate-limit events
  - Sniff backend used (browser-use, agent-browser, manual HAR)
- R5. The sniff report must include truncated response samples for each unique response shape so a reviewer can verify generated schemas match actual API behavior. Truncation rules: JSON/text responses include first 2KB or 100 lines (whichever is smaller); binary responses (images, protobuf, etc.) are skipped with a metadata note (content type, size).

**Raw Evidence Archival — Phase 1**

- R6. The raw capture file (HAR or enriched capture JSON) must be archived in `discovery/`. To control size, strip response bodies from the HAR and rely on the truncated samples in the sniff report (R5) for schema evidence.
- R7. The deduplicated URL/path list (`sniff-unique-paths.txt` or equivalent) must be archived in `discovery/`.

**Crowd-Sniff Discovery Report — Phase 2 (after PR #67)**

- R8. At the end of the crowd-sniff flow, the skill must produce a structured crowd-sniff report at `$API_RUN_DIR/discovery/crowd-sniff-report.md` containing:
  - npm packages analyzed (name, version, download count, recency)
  - GitHub repos searched (query used, repos matched, freshness)
  - Endpoints discovered with source tier and frequency (seen in N sources)
  - Base URL candidates discovered and which was selected
  - Auth patterns detected (API key, bearer, OAuth)
- R9. The binary's `crowd-sniff --json` provenance output must be archived in `discovery/`. Raw SDK tarballs and GitHub responses are NOT archived (too large, reproducible via re-run).

## Success Criteria

- A sniff-derived CLI's manuscripts contain enough information to reproduce the sniff: a new developer can see what pages were visited, what endpoints were found, and re-run or extend the discovery.
- A reviewer can trace any endpoint in the generated spec back to observed traffic evidence (response sample or HAR entry).
- `publish validate` manuscripts check (warn-only) passes for discovery-derived CLIs. When discovery/ is present, it is included; when absent (non-discovery CLI or pre-discovery run), the check does not flag it.

## Scope Boundaries

- Not adding provenance retroactively to existing CLIs.
- Not changing the research brief or shipcheck formats.
- Not storing full response bodies in the HAR — strip bodies and use truncated samples in the report instead.
- Not changing `publish validate` manuscripts check from warn-only to required.
- Not designing for hypothetical future discovery methods beyond sniff and crowd-sniff.
- Phase 2 (crowd-sniff) does not begin until PR #67 is merged.

## Key Decisions

- **Single `discovery/` directory, not per-method dirs**: Both sniff and crowd-sniff write to `discovery/` with method-prefixed filenames. One archive step handles both. The directory structure is ready for both methods from Phase 1; Phase 2 adds content only.
- **Phased delivery**: Phase 1 (sniff) lands independently. Phase 2 (crowd-sniff) gates on PR #67. Avoids coupling to unmerged work.
- **Skill-generated reports, binary provenance deferred**: Phase 1 reports are skill-generated markdown. Binary `--json` provenance output (structured per-endpoint extraction metadata) is deferred to planning — the skill has enough context from orchestrating the sniff to write a useful report without it.
- **Content-aware truncation for response samples**: JSON/text first 2KB or 100 lines; binary responses skipped with metadata note. Avoids arbitrary line-count truncation that fails for non-text.
- **HAR archived with bodies stripped**: Raw HAR structure preserved for reproducibility (URLs, methods, status codes, headers) but response bodies removed to control size. Truncated samples in the report serve the auditability need.

## Outstanding Questions

### Deferred to Planning

- [Affects R3][Technical] Does `publish package` already copy the full `.manuscripts/<run-id>/` tree recursively, or does it need code changes to include `discovery/`?
- [Affects R2][Technical] `ArchiveRunArtifacts` in `publish.go` uses a hardcoded pairs list. Needs a `DiscoveryDir` pair added, plus `RunDiscoveryDir()` and `ArchivedDiscoveryDir()` path functions in `paths.go`.
- [Affects R6][Technical] What tool or code strips response bodies from HAR before archival? Could be a `jq` one-liner in the skill or a utility in the binary.
- [Phase 2][Needs research] Does `crowd-sniff --json` (PR #67) already emit enough structured output for the crowd-sniff report, or does it need a provenance mode?
- [Phase 2][Technical] Binary `--json` provenance output for both sniff and crowd-sniff — schema design deferred until skill-generated reports prove insufficient.

## Next Steps

`/ce:plan` for structured implementation planning (Phase 1: sniff provenance)
