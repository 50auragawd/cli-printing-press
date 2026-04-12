# feat: README/SKILL narrative enrichment + generated SKILL.md

**Date:** 2026-04-12
**Branch:** `claude/cli-documentation-generation-6WK0e`
**Scope:** machine (generator + pipeline + templates)
**Commit scope:** `cli`

## Problem

The printing press currently under-communicates what makes each printed CLI valuable:

1. **README under-sells novel features.** The `## Unique Features` section is a flat bullet list of `` `command` — one-line description ``. No examples, no "why it matters," no narrative grouping. Compared to a hand-optimized README (yahoo-finance), the generic render reads as scaffolding.
2. **Root `--help` is bare.** `root.go.tmpl` emits only a `Short` (`"Manage {name} resources via the {name} API"`). Agents running the printed CLI with zero context have to do discovery via subcommand `--help` calls to find what's novel or that agent-mode flags exist. Token-wasteful.
3. **No SKILL.md is generated.** The downstream `generate-skills` script in `mvanhorn/printing-press-library` synthesizes one deterministically. It lacks the research context available at generation time (brief, competitor analysis, novel features), so every generated skill is thin boilerplate that makes agents re-discover the CLI instead of knowing its capabilities upfront.

## Bugs uncovered along the way

A side-by-side of a generic vs. optimized README exposed four machine bugs that must be fixed alongside the feature work:

- **B1** `<!-- HELP_OUTPUT -->` / `<!-- DOCTOR_OUTPUT -->` / `<!-- VERSION_OUTPUT -->` markers are left unreplaced in the final README. `internal/generator/readme_augment.go:15` handles replacement but isn't reliably invoked after tier-1 capture completes during the main generate path.
- **B2** `## Cookbook` block (`internal/generator/templates/readme.md.tmpl:224–246`) hard-codes commands like `sync`, `search "query"`, `export --format jsonl` that don't exist in most printed CLIs. Hallucinated boilerplate.
- **B3** `## Configuration` → `Environment variables:` renders as an empty section when `.Auth.EnvVars` is empty (e.g., cookie/composed auth). Dangling header.
- **B4** `firstResource` helper in Output Formats + Cookbook can produce nonsense commands (e.g., `autocomplete list` when `autocomplete` has no `list` endpoint). Replace with a helper that picks a resource+subcommand that actually exists.

## Non-goals

- A "Why this CLI?" comparison table. Considered and **intentionally not pursued** — see "Rejected: competitor comparison table" below.
- Changing per-command `--help` output. Currently driven by spec descriptions — leave alone.
- Updating the downstream `generate-skills` script. That lives in `mvanhorn/printing-press-library` and is a separate PR.

## Rejected: competitor comparison table

An earlier version of this plan deferred a `## Why this CLI?` section that would render a table grouping competitors by archetype (library / tiny CLI / MCP server) with per-bucket weakness lines. After shipping the narrative enrichment, we re-evaluated and decided to **drop it permanently** rather than defer.

**Why the current approach already covers the need:**

- The bold `Headline` + `ValueProp` state the differentiator positively in the first paragraph.
- Grouped `## Unique Features` with `Example` and `WhyItMatters` demonstrates the novel capabilities with copy-pasteable commands — readers see what they can do, not what others can't.
- `## Sources & Inspiration` already credits the real competitors (name, stars, language) generously, without calling out their gaps.

**Why adding a comparison table is a bad trade:**

- **Decay risk.** A table claiming "tool X has no JSON output" becomes wrong the moment X adds it, and we have no signal to regenerate. Positive differentiation doesn't decay because it describes our CLI, not theirs.
- **Reputation risk.** Auto-generated "what's missing" lines about other people's open-source projects, in a README that also credits them, is a jarring tonal shift. A single LLM hallucination about a real project is visibly wrong in a public library.
- **New failure mode.** Positive differentiation degrades gracefully — a CLI with few novel features just has a shorter Unique Features block. A comparison table needs competitor classification + weakness synthesis to work at all.
- **Marketing vs. reference docs.** Comparison tables are marketing copy that works when a human author *knows* the ecosystem. Machine-generated marketing copy about other people's work is a fundamentally different risk profile than machine-generated docs about our own CLI.

**When a future CLI needs to position against a named incumbent**, that's a narrative field the absorb skill can author as prose in `ValueProp` ("unlike existing Python wrappers, this ships as a single Go binary with no runtime deps") — no table infrastructure needed. That path already works today.

## Design decisions (locked)

| # | Decision | Rationale |
|---|---|---|
| Q1 | Minimal novel-feature enrichment: add `Example`, `WhyItMatters`, `Group` | Richer than flat bullets; less failure-prone than full recipes |
| Q2 | Add compact root `Long` with top 2–3 novel features + `--agent` pointer + `doctor` pointer | Closes the CLI-only agent discovery gap without bloating per-command help |
| Q3 | SKILL.md lands at `library/<category>/<api>/SKILL.md` (sibling to README) | First-class artifact, visible in PRs, downstream becomes copy-not-synthesize |
| Q4 | Hybrid: deterministic template structure + LLM-authored narrative fields | Uses already-running absorb pass; reproducible on rebuild via `research.json` |
| Q5 | Loose target shape (yahoo-finance SKILL.md is a data point, not gold standard) | Pattern derived from the optimized README's groupings |
| Q6 | Medium scope: fix the 4 bugs, enrich README with narrative, add SKILL.md, add root `Long` | Lower-risk than wide scope; competitor-table option rejected outright (see "Rejected" section) |
| — | Drop `## Cookbook` section entirely | Quick Start + grouped Unique Features with examples do the job; optimized README has no Cookbook |
| — | LLM-author trigger phrases per CLI | Domain verbs vary (finance: "quote X"; media: "play X"); template verbs are too generic |
| — | No silent fallbacks for LLM failures in absorb | Printing press already hard-depends on LLM; narrative rides on existing call. Validate JSON, retry once on parse failure, fail absorb on second failure. |

## Schema changes

**`internal/pipeline/research.go`**

```go
// Enriched
type NovelFeature struct {
    Name         string   `json:"name"`
    Command      string   `json:"command"`
    Description  string   `json:"description"`
    Rationale    string   `json:"rationale"`
    Aliases      []string `json:"aliases,omitempty"`
    // NEW (absorb LLM)
    Example      string   `json:"example,omitempty"`
    WhyItMatters string   `json:"why_it_matters,omitempty"`
    Group        string   `json:"group,omitempty"`
}

// NEW
type ReadmeNarrative struct {
    Headline        string             `json:"headline"`         // bold value prop
    ValueProp       string             `json:"value_prop"`       // 2–3 sentence expansion
    AuthNarrative   string             `json:"auth_narrative"`   // API-specific
    QuickStart      []QuickStartStep   `json:"quickstart"`       // real example commands
    Troubleshoots   []TroubleshootTip  `json:"troubleshoots"`
    // SKILL-specific
    WhenToUse       string             `json:"when_to_use"`
    Recipes         []Recipe           `json:"recipes"`
    TriggerPhrases  []string           `json:"trigger_phrases"`
}

type QuickStartStep struct {
    Command string `json:"command"`
    Comment string `json:"comment"`
}

type Recipe struct {
    Title       string `json:"title"`
    Command     string `json:"command"`
    Explanation string `json:"explanation"`
}

type TroubleshootTip struct {
    Symptom string `json:"symptom"`
    Fix     string `json:"fix"`
}

// Added to ResearchResult
type ResearchResult struct {
    // ... existing fields ...
    Narrative *ReadmeNarrative `json:"narrative,omitempty"`
}
```

All new fields optional at the JSON layer so research.json from previous runs still loads; templates guard with `{{if .Narrative}}…{{end}}`.

## Phases

### Phase 1 — Schema + absorb prompt
- Extend `NovelFeature` and add `ReadmeNarrative` structs in `internal/pipeline/research.go`.
- Extend absorb prompt (file TBD during implementation — search `internal/pipeline/absorb*.go` or wherever novel features are authored) to emit enriched features plus the narrative object as one JSON blob.
- Strict JSON validation. One retry on parse failure. Hard-fail on second failure.
- Persist in `research.json`.

### Phase 2 — Bug fixes
- **B1:** Ensure `AugmentREADME` runs after tier-1 dogfood capture during the main generate flow, not only during publish/emboss. Add test asserting no `<!-- *_OUTPUT -->` markers remain in final README.
- **B2:** Delete `## Cookbook` block from `readme.md.tmpl:224–246`.
- **B3:** Guard `Environment variables:` with `{{if .Auth.EnvVars}}`.
- **B4:** Replace `firstResource` uses in Output Formats with a helper that returns a resource+subcommand pair that actually exists (or omit the block).

### Phase 3 — README template rewrite
`internal/generator/templates/readme.md.tmpl` — restructure top sections, consuming `Narrative` with graceful fallback to current generic content when `Narrative` is absent.

Section order:
1. `# {{humanName .Name}} CLI`
2. `**{{.Narrative.Headline}}**` (fallback `{{.Description}}`)
3. `{{.Narrative.ValueProp}}` (optional)
4. `## Install` (unchanged)
5. `## Authentication` — `{{.Narrative.AuthNarrative}}` if present, else current auth-branch flow
6. `## Quick Start` — `{{range .Narrative.QuickStart}}` if present, else current auth-branch fallback
7. `## Unique Features` — grouped by `.Group` when any features have groups; otherwise flat. Each feature: `` **`{{.Command}}`** — {{.Description}}`` + optional `Example` code block + optional `WhyItMatters` italic line
8. `## Usage` + `<!-- HELP_OUTPUT -->`
9. `## Commands` (unchanged)
10. `## Output Formats` (bug-fixed)
11. `## Agent Usage` (unchanged)
12. `## Use as MCP Server` (unchanged)
13. **`## Cookbook` removed**
14. `## Health Check` + `<!-- DOCTOR_OUTPUT -->`
15. `## Configuration` (env-var guarded)
16. `## Troubleshooting` — `.Narrative.Troubleshoots` if present, else generic
17. `## Sources & Inspiration` (unchanged)

### Phase 4 — Root `--help` Long
`internal/generator/templates/root.go.tmpl` — add `Long` field on root cobra command. Template pulls `Headline` + top 3 entries of `NovelFeaturesBuilt` + `--agent` hint + `doctor` pointer. Degrades to a 2-line generic `Long` if `Narrative` and `NovelFeaturesBuilt` are absent.

Plumb `Narrative` and `NovelFeaturesBuilt` into the CLI template data struct (already plumbed for the README generator — extend).

### Phase 5 — SKILL.md generation
**New:** `internal/generator/templates/skill.md.tmpl`.

Sections (order):
1. Frontmatter: `name`, `description` (derived from `Headline` + LLM-authored `TriggerPhrases`), `argument-hint`, `allowed-tools`, `metadata.openclaw` install manifest — all deterministic.
2. `# {{humanName .Name}} — Printing Press CLI`
3. `{{.Narrative.ValueProp}}`
4. `## When to Use This CLI` — `{{.Narrative.WhenToUse}}`
5. `## Unique Capabilities` — iterate `NovelFeaturesBuilt`; each feature renders as `### \`{{.Command}}\`` + description + italic `WhyItMatters` + fenced `Example`
6. `## Command Reference` — deterministic from resource graph (one line per endpoint)
7. `## Recipes` — iterate `Narrative.Recipes` (LLM-authored)
8. `## Auth Setup` — deterministic branch on `.Auth.Type` (shared logic with README)
9. `## Agent Mode` — explicit `--agent` flag expansion
10. `## Exit Codes` — deterministic table
11. `## Argument Parsing` — deterministic three-branch parser (help / install / else)
12. `## Installation` — deterministic go install + MCP install blocks

### Phase 6 — Publish-flow integration
`internal/pipeline/publish.go` around line 115 (next to `writeCLIManifestForPublish`):

```go
if err := writeSkillForPublish(ctx, libDir, genData); err != nil {
    return fmt.Errorf("writing SKILL.md: %w", err)
}
```

Renders `skill.md.tmpl` to `library/<category>/<api>/SKILL.md`. Non-fatal: logs warning and continues if render fails — a missing SKILL.md is better than a failed publish.

### Phase 7 — Tests
- `internal/generator/readme_augment_test.go` — assert no `<!-- *_OUTPUT -->` markers remain in a fully-generated README fixture.
- `internal/generator/skill_render_test.go` (new) — golden-file test exercising: auth-type branches (api_key, oauth2, cookie, composed, bearer, none), grouped vs. flat novel features, empty novel features, absent narrative.
- `internal/generator/readme_template_test.go` — cases: no novel features → section omitted; no narrative → fallback rendering; no env vars → no dangling header; cookbook block absent.
- `internal/pipeline/novel_features_matcher_test.go` — update fixtures to include `Example`, `WhyItMatters`, `Group`.
- Integration test (if one exists for publish flow) asserting `SKILL.md` is written to `library/<cat>/<api>/SKILL.md`.
- `go test ./...` and `gofmt -w ./...` before commit.

## Commit plan (on `claude/cli-documentation-generation-6WK0e`)

1. `fix(cli): unreplaced help/doctor/version markers in generated README` (B1)
2. `fix(cli): remove hallucinated cookbook block and placeholder resource examples` (B2, B4)
3. `fix(cli): hide empty environment variables section in README` (B3)
4. `feat(cli): enrich novel-feature schema with example, why-it-matters, group` (Phase 1 schema slice)
5. `feat(cli): absorb LLM pass authors README/SKILL narrative from brief` (Phase 1 prompt slice)
6. `feat(cli): rewrite README template with narrative headline, grouped unique features, API-specific auth and troubleshooting` (Phase 3)
7. `feat(cli): generate root --help Long with top novel features and agent-mode pointer` (Phase 4)
8. `feat(cli): generate SKILL.md alongside README at publish time` (Phases 5 + 6)

Each commit is separately shippable and separately revertable. Tests land with the commit that introduces the behavior they cover.

## Downstream coordination (separate PR, separate repo)

Coordinate change to `mvanhorn/printing-press-library`'s `generate-skills` script:
- If `library/<cat>/<api>/SKILL.md` exists → copy to `plugin/skills/pp-<api>/SKILL.md`.
- Else → fall back to current synthesis (back-compat for CLIs published before this change).

Not part of this plan; noted for follow-through.

## Risk / rollback

- **Risk:** Absorb prompt changes could regress existing novel-features quality. **Mitigation:** snapshot existing `research.json` outputs from recent runs as test fixtures; new prompt must produce equivalent-or-better novel features on those fixtures.
- **Risk:** Template changes break existing printed CLIs that regenerate. **Mitigation:** all new template fields are guarded with `{{if .Narrative}}` — old `research.json` without narrative still renders cleanly.
- **Rollback:** Each commit is independently revertable. Template changes revert cleanly; schema additions are backward-compatible at JSON layer.

## Success criteria

1. A regenerated printed CLI's README has a bolded headline, a grouped `## Unique Features` section with examples, and no `<!-- *_OUTPUT -->` / hallucinated-cookbook / dangling-header artifacts.
2. `<name>-pp-cli --help` shows a `Long` description naming the top 2–3 novel commands and pointing at `--agent` + `doctor`.
3. `library/<cat>/<api>/SKILL.md` exists after publish with frontmatter, unique-capabilities block, recipes, command reference, and trigger phrases matching the API's domain.
4. Downstream `generate-skills` script (in follow-up PR) can copy instead of synthesize.
5. `go test ./...`, `golangci-lint run ./...`, and `gofmt -w ./...` all clean.
