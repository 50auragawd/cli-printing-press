---
name: printing-press
description: Generate the GOAT CLI for any API. 5-phase loop with Non-Obvious Insight Review and Ship Readiness Assessment, deep competitor research, complex body field handling, and before/after scoring delta.
version: 1.2.0
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - WebFetch
  - WebSearch
  - AskUserQuestion
  - Agent
---

# /printing-press

Generate the best CLI that has ever existed for any API. Five mandatory phases. Non-Obvious Insight Review + Ship Readiness Assessment. No shortcuts.

```
/printing-press Notion
/printing-press Plaid payments API
/printing-press --spec ./openapi.yaml
/printing-press Discord codex          # Codex mode: offload code generation to save Opus tokens
/printing-press emboss ./discord-cli   # Second pass: improve an existing CLI
```

## Emboss Mode (Second Pass)

**Emboss is opt-in.** It NEVER runs automatically. It runs when:
1. The user explicitly types `/printing-press emboss <dir>`
2. The user selects "Yes, run emboss" from the Phase 5.9 prompt after a main run

If the user did NOT request emboss, do NOT mention it, do NOT run it, do NOT show emboss reports.

When the user's arguments start with `emboss`, this is NOT a from-scratch run. The CLI already exists. Run a 30-minute improvement cycle.

```
if the user's arguments start with "emboss":
  EMBOSS_MODE = true
  EMBOSS_DIR = first argument after "emboss"
  Verify the directory exists and contains a Go CLI (check for cmd/ and internal/cli/)
else:
  EMBOSS_MODE = false (default - normal generation)
```

### The Emboss Cycle (6 steps, ~30 minutes)

**Step 1: AUDIT (5 min)** - Get a baseline without changing anything.

```bash
cd ~/cli-printing-press && ./printing-press emboss --dir <cli-dir> --spec <spec-path> --audit-only
```

Read the output. Note the scorecard score, verify pass rate, data pipeline status, and command count. This is the "before" snapshot.

Also read:
- The CLI's README for what commands exist
- Any Phase 0-5 artifacts in `docs/plans/` for this API
- The CLI's `internal/cli/root.go` to catalog registered commands

**Step 2: RE-RESEARCH (10 min)** - What's changed since v1?

This is NOT a full Phase 0 redo. Run targeted searches:

1. **WebSearch**: `"<API name>" CLI tool 2026` (any new competitors since v1?)
2. **WebSearch**: `"<API name>" "I wish" OR "I built" site:reddit.com OR site:news.ycombinator.com` (new pain points?)
3. Check npm: has anyone published a new CLI for this API?
4. Check if the API spec has been updated (new endpoints?)

Output: a "what's new" briefing (5-10 bullet points, not a full research document).

**Step 3: GAP ANALYSIS (5 min)** - What are the top 5 improvements?

Compare the audit baseline + re-research against what's possible. Score each potential improvement:

| Improvement | User Impact (1-5) | Score Impact (1-5) | Effort (1-5, 5=easy) | Total |
|------------|-------------------|-------------------|---------------------|-------|

Pick the top 5. Present to the user for approval before building.

Common improvement categories:
- Fix broken commands (from verify failures)
- Add missing workflow commands (from re-research)
- Improve data layer (add tables, fix sync, add FTS5)
- Polish README (add cookbook, fix examples)
- Add new endpoints (from spec updates)

**Step 4: IMPROVE (15 min)** - Build the top 5.

For each approved improvement:
1. Implement it (delegate to Codex if codex mode)
2. Run `go build && go vet` to verify compilation
3. Commit atomically: `feat(<api>): <improvement description>`

**Step 5: RE-VERIFY (5 min)** - Prove it worked.

```bash
cd ~/cli-printing-press && ./printing-press emboss --dir <cli-dir> --spec <spec-path> --audit-only
```

Compare the new numbers to the baseline from Step 1.

**Step 6: REPORT** - Tell the user the delta.

```
EMBOSS COMPLETE: <api>-cli
  Scorecard: <before> -> <after> (+<delta>)
  Verify:    <before>% -> <after>% (+<delta>%)
  Commands:  <before> -> <after> (+<delta>)
  Pipeline:  <before> -> <after>
  Top improvements: <list>
```

### Emboss Phase Gate

The emboss is successful if:
- Scorecard improved by at least 3 points
- Verify pass rate improved or stayed the same
- No new critical failures introduced
- All improvements compile and pass `go vet`

If verify pass rate DECREASED, something broke. Revert the last improvement and investigate.

---

## Codex Mode (Opt-In)

Add `codex` to the command to offload code generation and fix application (Phase 4, 4.5, 4.9, 5.7) to Codex CLI. Claude stays the brain (research, planning, scoring, review). Codex does the hands (writing Go code). Saves ~60% Opus tokens per run.

**Default is OFF.** Standard Opus mode runs unless you explicitly type `codex`.

### Mode Detection

```
if the user's arguments contain "codex" or "--codex":
  CODEX_MODE = true
  Verify: command -v codex >/dev/null 2>&1
  If codex not installed: print "Codex CLI not found - running standard mode." and set CODEX_MODE = false
  Guard: if $CODEX_SANDBOX or $CODEX_SESSION_ID is set, print "Already inside Codex sandbox" and set CODEX_MODE = false
else:
  CODEX_MODE = false (default)
```

### Codex Delegation Pattern

When CODEX_MODE is true and a task is pure code generation (writing a Go file, applying a fix):

1. **Claude assembles the prompt** with: task description, exact files to modify, current code context (paste real code), expected change in plain English, conventions from the codebase, and constraints (no git, no PRs, <200 lines, run go build at end)

2. **Write prompt and delegate:**
```bash
CODEX_PROMPT="TASK: [1-sentence description]

FILES TO MODIFY:
- [exact paths]

CURRENT CODE:
[paste relevant functions/signatures from codebase]

EXPECTED CHANGE:
[plain English description of the diff]

CONVENTIONS:
- [commit style, import patterns, error handling from the codebase]

CONSTRAINTS:
- Do NOT run git commit, git push, or git add. The sandbox blocks .git writes.
- Do NOT modify files outside the listed paths.
- Keep changes under 200 lines.

VERIFY: After changes, run: go build ./... && go vet ./..."

cd ~/cli-printing-press && echo "$CODEX_PROMPT" | codex exec --yolo -
```

3. **Claude reviews the result:** Before anything else, verify the target file still exists and is non-empty (`wc -l <file>`). Codex can delete or empty files instead of rewriting them — if the file is gone or empty, that's an immediate failure. Then verify: in-scope changes, compiles (`go build && go vet`). If lint/format fails, auto-fix.

4. **On failure:** Fall back to Claude for that task. A deleted/emptied file counts as a failure. Track consecutive failures — after 3, disable Codex for remaining tasks.

### What Gets Delegated vs What Stays on Claude

| Delegated to Codex (code generation) | Stays on Claude (reasoning) |
|---|---|
| Writing store.go domain tables | Phase 0-1: Research, prediction engine |
| Writing workflow commands (sync, search, sql, etc.) | Phase 0.7: Architecture decisions |
| Writing insight commands (health, trends, etc.) | Phase 3: Non-Obvious Insight Review |
| Applying scorecard fixes (dead code, wiring flags) | Phase 4.7: Proof of Behavior verification |
| Applying reviewer fixes from Phase 4.9 | Reviewer dispatch, fix prioritization, acceptance decision |
| README cookbook section | Phase 5: Ship Readiness Assessment |
| Fix cycle patches (5-50 lines each) | Phase 5.5: Live API Testing |

## Prerequisites

- Go 1.21+ installed
- The printing-press repo at `~/cli-printing-press`
- Build binary if missing: `cd ~/cli-printing-press && go build -o ./printing-press ./cmd/printing-press`

## Phase 0.1: API KEY AUTO-DETECTION

Before asking the user for anything, silently check if a token is already available:

```
1. Check common env vars for the target API:
   - GitHub: $GITHUB_TOKEN, $GH_TOKEN, or run `gh auth token` if gh CLI is installed
   - Discord: $DISCORD_TOKEN, $DISCORD_BOT_TOKEN
   - Linear: $LINEAR_API_KEY
   - Notion: $NOTION_TOKEN
   - Stripe: $STRIPE_SECRET_KEY (read-only test key only)
   - Generic: $API_KEY, $API_TOKEN

2. If a token is found:
   Use AskUserQuestion to ask:
   "Found a [API_NAME] token in $[ENV_VAR]. Use it for read-only live testing at the end?"
   Options:
   - "Yes, use it" (read-only GETs only, never creates/updates/deletes)
   - "No, skip live testing" (use dry-run and mock validation only)

   **WAIT for the user's answer before proceeding.** Do NOT continue to Phase 0 until answered.

3. If no token found:
   Use AskUserQuestion to ask:
   "No [API_NAME] token detected. Want to provide one for live testing?"
   Options:
   - "I'll set it up" (user will paste or export the token, then you re-check)
   - "Skip, no live testing" (proceed without, use dry-run validation only)

   **WAIT for the user's answer before proceeding.** Do NOT continue to Phase 0 until answered.
```

The key insight: **detect first, ask permission second, WAIT for the answer.** Don't barrel ahead into research while the user is still deciding. The AskUserQuestion tool blocks until they respond.

## How This Works

Every run produces the GOAT CLI through 9 mandatory phases + 7 comprehensive plan documents:

```
PHASE 0 -> PHASE 0.5 -> PHASE 0.7 -> PHASE 0.8 -> PHASE 0.9 -> PHASE 1 -> PHASE 2 -> PHASE 3 -> PHASE 4 -> PHASE 4.5 -> PHASE 4.6 -> PHASE 4.8 -> PHASE 4.9 -> PHASE 5
(3-5m)     (2-3m)       (15-25m)     (5-8m)     (1-2m)     (5-8m)     (5-10m)    (10-20m)      (2-3m)       (5-10m)
Visionary  Workflows    Prediction   Research   Generate   Audit      Build      Dogfood       Final        Agent
Research   (commands)   Engine       (specs)    (code)     (review)   (fixes)    Emulation     Quality      Readiness
                        (data layer)                                             (spec-test)   Score        Review
```

Total expected time: 50-95 minutes. Phase 4.5 tests every command against spec-derived mocks.

**7 Plan Artifacts Per Run:**

Every phase gate produces a comprehensive plan document in `~/cli-printing-press/docs/plans/`:

```
Phase 0   -> <today>-feat-<api>-cli-visionary-research.md
Phase 0.5 -> <today>-feat-<api>-cli-power-user-workflows.md
Phase 0.7 -> <today>-feat-<api>-cli-data-layer-spec.md
Phase 1   -> <today>-feat-<api>-cli-research.md
Phase 3   -> <today>-fix-<api>-cli-audit.md
Phase 4   -> <today>-fix-<api>-cli-goat-build-log.md
Phase 4.5 -> <today>-fix-<api>-cli-dogfood-report.md
```

Each artifact chains into the next. **Read the previous phase's artifact before starting the next phase.**

**The quality bar:** Peter Steinberger's gogcli is the 10/10 reference. Every generated CLI is scored against it TWICE - once during the Non-Obvious Insight Review to find gaps, once in the Ship Readiness Assessment to prove improvement. The delta is the proof of work.

**Grade thresholds (10 dimensions, 100 max):**
- **Grade A:** 80+/100 (80%)
- **Grade B:** 65-79/100 (65-79%)
- **Grade C:** 50-64/100 (50-64%)

---

## Artifact Writing: Plan Generation at Each Phase Gate

At the end of each phase, write a comprehensive plan document. This is NOT optional - the artifacts ARE the product.

**Option A: /ce:plan is available (compound-engineering plugin installed)**

Try to invoke the `compound-engineering:ce:plan` skill. If it exists, use it:

```
Skill tool: compound-engineering:ce:plan
Args: "<phase description with all research gathered so far>"
```

The /ce:plan skill produces a full plan document with frontmatter, analysis, acceptance criteria, and sources. Pass it all the research from this phase as the feature description.

**Option B: Built-in plan writer (fallback when compound-engineering is NOT installed)**

If /ce:plan is not available, write the artifact yourself with this structure:

```markdown
---
title: "<Phase Name>: <API> CLI"
type: feat
status: active
date: <today>
phase: "<phase number>"
api: "<api name>"
---

# <Phase Name>: <API> CLI

## Overview
[2-3 paragraph executive summary of what this phase discovered/decided]

## Analysis
[Full analysis with tables, scores, evidence URLs, and reasoning]
[Every claim backed by evidence - WebSearch URLs, star counts, API docs]
[Scoring breakdowns showing how each number was computed]

## Decisions
[What was decided and WHY - rationale for each decision]
[What was rejected and WHY]

## Concrete Outputs
[SQL schemas, command definitions, sync strategies - real code, not pseudocode]
[Every output validated against the actual API]

## Acceptance Criteria
- [ ] [Measurable criteria for this phase's outputs]

## Sources
- [URLs, file paths, competitor repos with star counts]
```

**CRITICAL:** The built-in writer must match /ce:plan depth:
- Full analysis, not bullet summaries
- Evidence with source URLs, not assertions
- Scoring breakdowns, not just final numbers
- Concrete SQL/code examples, not pseudocode
- Validation proof ("I verified the API supports ?after= filtering"), not assumptions
- 200+ lines minimum per artifact

---

## Workflow: `--spec` shortcut

When the user provides `--spec <path-or-url>`, skip Phase 1 spec search (spec is provided). STILL run competitor research (Steps 1.2-1.5). Run all other phases.

## Workflow: Natural Language (Primary)

When the user provides an API name, run ALL five phases.

### Step 0: Parse intent and check known specs

Extract the API name. Optionally check `~/cli-printing-press/skills/printing-press/references/known-specs.md` for a cached spec URL.

If found in registry: note the URL as a hint for Phase 1, but STILL run full research.
If not found: Phase 1 searches for the spec. This is the normal path - most APIs won't be in the registry.

**The registry is a speed shortcut, not a gate.** Never refuse to run because an API isn't in the registry. Never hard-block because the registry says "GraphQL" or "Skipped." Phase 1 discovers the spec type dynamically.

---

# PHASE 0: VISIONARY RESEARCH

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

Before generating any CLI, understand what a thoughtful developer would build - not just what the OpenAPI spec says.

### Step 0a: API Identity & Domain Understanding

Understand what this API IS:

1. **WebFetch** the API's developer docs landing page
2. **WebSearch**: `"<API name>" developer documentation overview`
3. Extract:
   - **Domain category:** messaging, payments, productivity, infrastructure, analytics
   - **Primary users:** Who uses this API? (e.g., "bot developers", "server admins")
   - **Core entities:** What are the main objects? (e.g., "guilds", "channels", "messages")
   - **Data profile:**
     - Write pattern: append-only, mutable, or event-sourced?
     - Volume: high (millions of records), medium, or low?
     - Real-time: does the API have webhooks/websockets/SSE?
     - Search need: high (users need to find things) or low?

### Step 0b: Usage Pattern Discovery

Discover what people ACTUALLY DO with this API:

**Community Research (run these in parallel):**
1. **WebSearch**: `"<API name>" CLI workflow site:reddit.com`
2. **WebSearch**: `"<API name>" automation script site:github.com`
3. **WebSearch**: `"<API name>" "I built" OR "I made" OR "my tool" site:reddit.com OR site:news.ycombinator.com`
4. **WebSearch**: `"<API name>" tutorial automation workflow 2025 2026`

**Pain Point Research:**
5. **WebSearch**: `"<API name>" API "pain point" OR "limitation" OR "workaround"`
6. **WebSearch**: `site:stackoverflow.com "<API name>" API rate limit OR pagination OR bulk`

From all research, identify the **top 5 usage patterns** ranked by evidence score:

| Source | Weight |
|---|---|
| Existing tool with 100+ stars | 3 points |
| Existing tool with 10-99 stars | 2 points |
| Reddit/HN post with 50+ upvotes | 2 points |
| Reddit/HN post with 10-49 upvotes | 1 point |
| Stack Overflow question with 10+ votes | 1 point |
| Blog post / tutorial | 1 point |
| GitHub issue on competitor | 1 point |
| Cross-platform appearance (same need on 2+ platforms) | +2 bonus |

Score >= 6: Strong evidence. Include in CLI.
Score 3-5: Moderate evidence. Consider as optional.
Score < 3: Weak evidence. Skip.

### Step 0c: Tool Landscape Discovery (The Discrawl Finder)

Find ALL tools for this API, not just API wrappers:

**Tier 1: Direct CLI Search** (existing Phase 1 does this too)
1. **WebSearch**: `"<API name>" CLI tool github`

**Tier 2: Non-Wrapper Tool Search** (CRITICAL - finds discrawl-class tools)
2. **WebSearch**: `"<API name>" sync OR archive OR mirror site:github.com`
3. **WebSearch**: `"<API name>" search engine OR analytics OR dashboard site:github.com`
4. **WebSearch**: `"<API name>" backup OR export OR migration site:github.com`
5. **WebSearch**: `"<API name>" monitor OR watcher OR alerting site:github.com`

**Tier 3: Ecosystem Search**
6. **WebSearch**: `awesome "<API name>" site:github.com`

**Tier 4: Market Landscape Search** (CRITICAL - finds the REAL competitive terrain)

The API wrapper is not the only competitor. Developers stack 3-4 tools. Find them all.

7. **WebSearch**: `"<API name>" CLI alternative OR replacement`
8. **WebSearch**: `"<API domain>" TUI OR terminal tool 2026` (use the domain from Step 0a, e.g., "git" for GitHub, "payments" for Stripe)
9. **WebSearch**: `best "<API domain>" workflow tool`
10. **WebSearch**: `"<API name>" "I switched to" OR "better than"`

Classify the landscape into lanes:
- **Forge/Platform CLIs** - direct API wrappers (gh, glab, stripe-cli)
- **Workflow Overlays** - higher-level tools on top (Graphite, Git Town)
- **Alternative UX** - rethink the domain interaction (lazygit, jj, Warp)

Most developers stack tools from multiple lanes. Our CLI should complement the incumbent, not compete head-on.

For each tool found, classify it:

| Type | Description | Example |
|---|---|---|
| **API Wrapper** | Translates HTTP to CLI flags | discli |
| **Data Tool** | Adds local persistence/search | discrawl |
| **Workflow Tool** | Orchestrates multi-step sequences | Stripe fixtures |
| **Environment Tool** | Runs local simulation | Supabase CLI |
| **Integration Tool** | Bridges to other systems | Zapier integration |

**The press should generate CLIs that compete with Data Tools and Workflow Tools, not just API Wrappers.**

### Step 0d: Workflow Analysis

From usage patterns (0b) and tool landscape (0c), identify multi-step workflows:

For each workflow, document:
- **Steps:** The sequence of API calls
- **Frequency:** How often users perform this
- **Pain point:** What makes this hard with the raw API
- **Proposed CLI feature:** What compound command would solve it

### Step 0e: Architecture Planning

Based on data profile and workflows, decide what the CLI needs:

| Data Profile | Architecture |
|---|---|
| High volume + search need | SQLite + FTS5 (Discord, Slack) |
| Transaction data + reconciliation | Local ledger with diff tracking (Stripe, Plaid) |
| Document data + offline editing | Local Markdown/JSON sync (Notion, Confluence) |
| Low volume + simple CRUD | Standard API wrapper is fine (most APIs) |

For each decision area (persistence, real-time, search, bulk, caching), document:
- **Need level:** High / Medium / Low
- **Decision:** What to use
- **Rationale:** Why

### Step 0f: Feature Ideation - "Next 5 Features for the World"

Score each feature idea on 8 dimensions (16-point max):

| Dimension | Weight | Scoring |
|---|---|---|
| **Evidence strength** | 3 | 3=existing tool 100+ stars, 2=Reddit/SO demand, 1=weak, 0=speculation |
| **User impact** | 3 | 3=most users feel this pain, 2=niche, 1=nice-to-have, 0=nobody asked |
| **Implementation feasibility** | 2 | 2=can generate template, 1=needs custom code, 0=major infrastructure |
| **Uniqueness** | 2 | 2=no existing tool, 1=improves on existing, 0=already well-served |
| **Composability** | 2 | 2=great with pipes/agents, 1=somewhat, 0=interactive-only |
| **Data profile fit** | 2 | 2=perfect fit, 1=possible, 0=wrong shape |
| **Maintainability** | 1 | 1=generated code supports it, 0=needs human maintenance |
| **Competitive moat** | 1 | 1=hard to replicate, 0=trivial |

Score >= 12: **Must-have.** Build it.
Score 8-11: **Should-have.** Include as optional.
Score < 8: **Won't-have.** Skip or future work.

### Step 0g: Write the Visionary Research Artifact

**Write** to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-visionary-research.md`:

```markdown
## Visionary Research: <API> CLI

### API Identity
- Domain: <category>
- Primary users: <who>
- Data profile: <write pattern>, <volume>, <realtime>, <search need>

### Usage Patterns (Top 5 by Evidence)
1. <pattern> (Evidence: X/10) - <what it needs>
2. ...

### Tool Landscape (Beyond API Wrappers)
- <tool> (<stars> stars): <what it does>
- ...

### Workflows
1. <name>: <steps> -> Proposed: `<api>-cli <command>`
2. ...

### Architecture Decisions
- Persistence: <decision> because <rationale>
- Real-time: <decision> because <rationale>
- Search: <decision> because <rationale>
- Bulk: <decision> because <rationale>
- Cache: <decision> because <rationale>

### Top 5 Features for the World
1. <feature> (Score: X/16) - <1-line description>
2. ...
```

### PHASE GATE 0

**STOP.** Verify ALL of these before proceeding:
1. API Identity documented with data profile
2. At least 3 usage patterns with evidence scores
3. Tool landscape includes non-wrapper tools (Tier 2 search done)
4. At least 2 workflows with proposed CLI features
5. Architecture decisions match data profile
6. Top 5 features scored and ranked

**Write Phase 0 Artifact:** Run the Artifact Writing plan generator (see top of skill) with all Phase 0 research as input. Write to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-visionary-research.md`. Include: API identity, data profile, usage patterns with evidence, tool landscape, architecture decisions, top 5 features with full scoring.

Tell the user: "Phase 0 complete: Domain: [category]. Data profile: [volume]/[realtime]/[search]. Found [N] non-wrapper tools. Top feature: [name] (score [X]/16). Architecture: [key decision]. Proceeding to power user workflows."

---

# PHASE 0.5: POWER USER WORKFLOWS

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

The generator produces API wrappers. Power users want workflow tools. This phase predicts what compound commands would make the CLI genuinely useful - the kind of features that make discrawl (12 commands) more valuable than a 316-command API wrapper.

### Step 0.5a: Classify the API Archetype

Based on Phase 0 research, classify the API:

| Archetype | Signal | Example Workflows |
|---|---|---|
| **Communication** | Messages, channels, threads | Archive, offline search, monitor keywords, export conversations |
| **Project Management** | Issues, tasks, sprints, states | Stale issues, orphan detection, velocity, burndown, standup, triage |
| **Payments** | Charges, subscriptions, invoices | Reconciliation, webhook replay, fixture flows, revenue reports |
| **Infrastructure** | Servers, deployments, logs | State sync, log tailing, deploy orchestration, health dashboards |
| **Content** | Documents, pages, blocks, media | Backup to local files, diff, template management, publish workflows |
| **CRM** | Contacts, deals, pipelines | Pipeline reports, stale deal alerts, activity timelines, bulk updates |
| **Developer Platform** | Repos, PRs, CI runs | PR triage, CI monitoring, release management, dependency audit |

### Step 0.5b: Generate 10-15 Workflow Ideas

For the identified archetype, brainstorm compound workflows. Ask:
- "What does a power user of this API wish they could do in one command?"
- "What multi-step task do people automate with scripts today?"
- "What reporting/hygiene/monitoring task requires manual effort?"
- "What would make an engineering manager's life easier?"

Each workflow should:
- Combine 2+ API calls into one operation
- Solve a real recurring problem
- Be expressible as a single CLI command with flags

### Step 0.5c: Validate Against API Capabilities

For each workflow idea, check:
1. Does the API have the required endpoints/fields?
2. Can the required data be queried/filtered?
3. Are write operations available (for mutation workflows)?
4. For GraphQL APIs: does the schema have the required types?

Drop workflows the API can't support.

### Step 0.5d: Rank by Impact

Score each workflow on:
- **Frequency**: How often would users run this? (daily=3, weekly=2, monthly=1)
- **Pain**: How painful is the manual alternative? (high=3, medium=2, low=1)
- **Feasibility**: How hard to implement? (easy=3, medium=2, hard=1)
- **Uniqueness**: Does any existing tool do this? (no=3, partial=2, yes=0)

### Step 0.5e: Select Top 5-7 for Implementation

These become **mandatory Phase 4 work items**. They are NOT optional polish. They are the PRODUCT.

### Step 0.5f: Naming Pass (User Outcomes, Not API Resources)

For each selected workflow, rename it from API-speak to user-speak. The name should complete this sentence: **"I need to check ___"**

| API-oriented (bad) | User-oriented (good) | Why |
|---|---|---|
| `actions-health` | `ci-health` or `flaky` | Users say "is CI flaky?" not "are actions healthy?" |
| `contributors` | `leaderboard` or `who-shipped` | The question being answered |
| `activity` | `standup` | The workflow it serves |

Rules:
- Names should be verbs or nouns a developer would type naturally
- If the incumbent has a name for this concept, use a different name (don't collide)
- Max 15 characters, no hyphens if possible
- Test: "Would an engineering manager type this without reading --help first?"

### PHASE GATE 0.5

**STOP.** Tell the user: "Identified [N] power-user workflows for [API name]. Top 5:
1. [name] - [one-line description] (score [X]/12)
2. ...
These will be built as real commands in Phase 4, alongside the API wrapper."

**Write Phase 0.5 Artifact:** Run the Artifact Writing plan generator (see top of skill) with all Phase 0.5 analysis as input. Write to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-power-user-workflows.md`. Include: API archetype, all 10-15 workflow ideas, validation results, full scoring table, top 7 with implementation notes.

---

# PHASE 0.6: FEATURE PARITY AUDIT

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

Before brainstorming novel features, catalog what the competition already ships. This is the missing link between research and build. Phase 0.5 brainstorms additive workflows. This phase checks if we can do what the incumbent already does.

### Step 0.6a: Feature Matrix

For the top 2 competitors by stars (from Phase 0/1 research):

| Feature | Competitor A | Competitor B | Ours | Classification |
|---------|-------------|-------------|------|----------------|

List EVERY command and feature they offer. Read their README, --help output, and documentation.

### Step 0.6b: Classify Each Feature

- **TABLE STAKES**: >50% of users expect any CLI for this API to have it.
  Examples: `issue create`, `page get`, git branch from issue, clean CRUD names (get/create/update/delete)
- **NICE-TO-HAVE**: Useful but not expected. Won't lose users if missing.
  Examples: interactive prompts, TUI mode, plugins
- **ANTI-SCOPE**: Genuinely out of scope with justification.
  Examples: full TUI, mobile app, GUI

**Classification rules:**
- If ANY competitor with >100 stars has it, it's TABLE STAKES unless you provide an explicit reason it's anti-scope.
- If users mention it in issues/Reddit with >10 upvotes, it's TABLE STAKES.
- "Complements the incumbent" is NOT a reason to skip a feature. Users don't want to install two CLIs.

### Step 0.6c: Table Stakes Become Phase 4 Mandatory Work

Every TABLE STAKES feature becomes a Phase 4 Priority 1 work item. They are built ALONGSIDE the data layer, not instead of it.

### PHASE GATE 0.6

**STOP.** Verify:
1. Feature matrix complete for top 2 competitors
2. Every feature classified as TABLE STAKES / NICE-TO-HAVE / ANTI-SCOPE
3. TABLE STAKES list has at least 3 items
4. Anti-scope items have explicit cost analysis ("What % of users need this? Does a competitor with >100 stars offer it?")

Tell the user: "Feature parity audit: [N] table-stakes features identified from [competitor names]. Top gaps: [list]. These will be built in Phase 4."

**Write Phase 0.6 Artifact:** Run the Artifact Writing plan generator with all Phase 0.6 analysis as input. Write to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-feature-parity-audit.md`. Include: full feature matrix, classifications with justification, table stakes list for Phase 4.

---

# PHASE 0.7: POWER USER PREDICTION ENGINE

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

The generator produces API wrappers. Power users want a local data layer - domain-specific SQLite tables, incremental sync, full-text search with domain filters, raw SQL access, and trend detection. This phase predicts that data layer from the API surface + social signals, WITHOUT looking at competitors (that's Phase 1's job).

**Read the Phase 0 and Phase 0.5 artifacts before starting this phase.**

### Step 0.7a: Entity Classification

Map every API resource into one of four types by reading the OpenAPI spec or Phase 0's entity list:

| Type | Signal | Example | Persistence Need |
|---|---|---|---|
| **Accumulating** | Grows over time, has timestamps, paginated lists | Messages, Issues, Audit Logs, Commits | SQLite table + incremental sync |
| **Reference** | Changes rarely, small cardinality, referenced by other entities | Users, Teams, Roles, Labels, Channels | SQLite table + periodic refresh |
| **Append-only** | Never edited, only created | Events, Webhooks, Notifications | SQLite table + tail command |
| **Ephemeral** | Short-lived, not worth persisting | OAuth tokens, Rate limit status, Gateway info | API-only, no persistence |

**Heuristics:**
- Has `created_at`/`timestamp` + paginated list endpoint -> Accumulating
- Referenced by 3+ other entities via `_id` fields -> Reference
- Has no UPDATE/PATCH endpoint -> Append-only
- No list endpoint or < 100 expected records -> Ephemeral
- Has `updated_at` or `modified_at` -> needs incremental sync cursor

**Output:** Entity classification table with type, estimated volume, update frequency, and key temporal field for ALL API resources.

### Step 0.7b: Social Signal Mining for Data Patterns

Find evidence of what data power users actually store locally. Run 7 parallel WebSearches:

1. **WebSearch**: `"<API name>" export OR backup OR archive site:github.com`
2. **WebSearch**: `"<API name>" SQLite OR database OR local site:github.com`
3. **WebSearch**: `"<API name>" analytics OR dashboard OR metrics site:github.com`
4. **WebSearch**: `"<API name>" "I wish" OR "would be nice" OR "feature request" data`
5. **WebSearch**: `"<API name>" offline OR search OR "full text" site:reddit.com OR site:news.ycombinator.com`
6. **WebSearch**: `"<API name>" trend OR pattern OR anomaly detection`
7. **WebSearch**: `"<API name>" graph OR visualization OR dependency`

**For each finding, extract:**
- What entities they store locally
- What queries they run (joins, aggregations, time filters)
- What temporal patterns they track (trends, anomalies, velocity)
- What cross-entity relationships they need

**Score using Phase 0 evidence framework.** Anything with score >= 6 informs the data layer.

### Step 0.7c: Data Gravity Scoring

Rank entities by how much value they'd have in a local SQLite database.

**Formula:** `DataGravity = Volume(0-3) + QueryFrequency(0-3) + JoinDemand(0-2) + SearchNeed(0-2) + TemporalValue(0-2)`

| Factor | 0 | 1 | 2 | 3 |
|---|---|---|---|---|
| **Volume** | < 100 records | 100-10k | 10k-1M | > 1M |
| **QueryFrequency** | Rarely queried | Monthly | Weekly | Daily |
| **JoinDemand** | No references | 1-2 entities reference it | 3-4 | 5+ |
| **SearchNeed** | No text fields | 1 text field | 2-3 text fields | Primary text content |
| **TemporalValue** | No time dimension | Created date only | Updated + trends | Core to time-series analysis |

**Thresholds:**
- Score >= 8: **Primary entity** - gets its own SQLite table with proper columns, FTS5 if text-heavy
- Score 5-7: **Support entity** - gets a simpler table
- Score < 5: **API-only** - no local persistence

Score EVERY entity from Step 0.7a. Show the full breakdown.

### Step 0.7d: Schema + Sync + Search Specification

For each Primary entity (score >= 8), produce:

**1. SQLite Schema:**
- Extract columns from the API's response schema (NOT just id + JSON blob)
- Include foreign key columns for joins (e.g., `channel_id`, `author_id`)
- Include the temporal field for sync cursors
- Add indexes on foreign keys and temporal fields
- Create FTS5 virtual table on text fields (title, description, content, body, name)
- Keep a `data JSON NOT NULL` column for the full API response

**2. Sync Strategy:**
- Identify the incremental sync cursor (timestamp field, snowflake ID, cursor pagination)
- **VALIDATE:** Check that the API supports filtering by this cursor - look for `since`, `after`, `updated_after`, `before` query params in the spec
- Determine batch size from API's max `limit` parameter
- Check if API has WebSocket/SSE/Gateway - note for tail command
- If the API doesn't support cursor filtering, fall back to full sync + local dedup

**3. Search Specification:**
- List which text fields to extract into FTS5
- Define domain-specific search filters as SQL WHERE clauses
- Map CLI flags to SQL: `--channel` -> `WHERE channel_id = ?`, `--author` -> `WHERE author_id = ?`
- **VALIDATE:** Confirm these fields actually appear in API list/get responses

**4. Compound Queries:**
- Define 3-5 cross-entity queries (e.g., "messages by author in channel in last N days")
- Validate join columns exist in both tables
- These become Phase 4 workflow commands that use local DB instead of live API

**5. Tail Strategy:**

| Method | When to Use |
|---|---|
| WebSocket/Gateway | API has it (Discord Gateway, Slack RTM) |
| SSE | API has it (GitHub, Linear webhooks) |
| REST Polling | Fallback - GET with ?since= cursor |

Decide which method this API supports. If WebSocket/SSE, the tail command should use it instead of REST polling.

### Step 0.7e: Write the Data Layer Specification Artifact

**Run the Artifact Writing plan generator** (see top of skill) with all Phase 0.7 analysis as input. Write to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-data-layer-spec.md`.

The artifact MUST include:
- Entity classification table for every API resource
- Data gravity scores with full breakdown per entity
- Complete SQLite schema (CREATE TABLE + CREATE INDEX + FTS5)
- Sync strategy with cursor validation proof
- Domain-specific search filters mapped to SQL WHERE clauses
- 3-5 compound cross-entity queries
- Tail strategy decision with justification
- Commands to build in Phase 4 Priority 0

### PHASE GATE 0.7

**STOP.** Verify ALL of these before proceeding:
1. Entity classification table with type and volume estimates for every API resource
2. At least 3 social signals with evidence scores >= 6
3. Data gravity scores computed, with >= 1 primary entity (score >= 8)
4. SQLite schema with proper columns (NOT generic JSON blobs) for each primary entity
5. FTS5 virtual tables for entities with text fields
6. Sync strategy with cursor fields validated against actual API filter params
7. Domain-specific search filters mapped to SQL WHERE clauses
8. At least 3 compound queries validated (joins work, columns exist)

Tell the user: "Phase 0.7 complete: [N] primary entities for SQLite ([list]), [M] compound queries validated. Sync via [cursor type]. FTS5 on [fields]. Key prediction: [most valuable data-layer feature]. Proceeding to product thesis."

---

# PHASE 0.8: PRODUCT THESIS

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

Before generating any code, articulate why someone would install this CLI. If you can't answer these five questions in one sentence each, the research phases missed something - go back.

### Step 0.8a: Answer Five Questions

1. **Who is this for?** (one sentence, specific persona)
   - Bad: "Developers who use the GitHub API"
   - Good: "Engineering managers who need cross-repo PR triage without a dashboard"

2. **What's the comparison table?** (us vs incumbent, 5 rows minimum)
   | Capability | Incumbent | Ours |
   |-----------|-----------|------|
   Fill this in with real capabilities from Phase 0 research.

3. **What's the HN headline?** (one sentence that makes a developer click)
   - Bad: "A new CLI for the GitHub API"
   - Good: "I built a GitHub CLI that finds stale PRs and lets you SQL query your repos offline"

4. **What's the name?**
   - DEFAULT: `<api>-pp-cli` (e.g., `notion-pp-cli`, `linear-pp-cli`, `stripe-pp-cli`)
   - The `-pp-` identifies it as a printing press product.
   - This is discoverable (`brew search notion` finds `notion-pp-cli`).
   - Creative names are allowed ONLY if the user explicitly requests one.
   - The printing press is a code generator, not a branding agency.

5. **What's the anti-scope and what does it cost?**

   For each anti-scope item, answer:
   - What % of potential users need this feature?
   - Does any competitor with >100 stars offer it?
   - If yes: this is NOT anti-scope, it's a backlog item. Move to Phase 4 Priority 1.

   VALID anti-scope: "Not a TUI" (no competitor offers one either)
   INVALID anti-scope: "Not a git integration" (the top competitor's killer feature - you just excluded the #1 reason people install it)

### Step 0.8b: Write the Product Thesis

Write one paragraph that combines the answers above. This paragraph should make a developer say "I need this." It goes in the README later.

### PHASE GATE 0.8

**STOP.** Verify:
1. All 5 questions answered with specific, non-generic answers
2. Product thesis paragraph written
3. Name set to `<api>-pp-cli` (default) unless user specified otherwise
4. Comparison table has at least one row where we clearly win

Tell the user: "Product thesis: [1-sentence pitch]. Name: [name]. Key differentiator: [comparison table winner]. Proceeding to deep research."

---

# PHASE 0.9: CHECK FOR PRIOR RESEARCH

Before starting Phase 1 research from scratch, check if the user already did research:

```bash
ls ~/cli-printing-press/docs/plans/*<api-name>* ~/docs/plans/*<api-name>* 2>/dev/null
```

If found:
1. **Read** every matching plan document
2. Extract: competitive landscape, user pain points, tool rankings, product positioning
3. **Skip redundant Phase 1 research** - if the prior plan already covers competitor analysis and demand signals, don't re-search. Focus Phase 1 on filling gaps (spec discovery, auth method, endpoint count).
4. Note which Phase 1 steps are already answered by prior research.

If not found: proceed to Phase 1 normally.

---

# PHASE 1: DEEP RESEARCH

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

Research the API landscape deeply. You need to understand the competitive terrain, user pain points, and strategic opportunity before generating anything.

### Step 1.1: Search for the API spec

Search for BOTH REST and GraphQL specs. Don't assume the API type - discover it.

1. **WebSearch**: `"<API name>" openapi spec site:github.com`
2. **WebSearch**: `"<API name>" openapi.yaml OR openapi.json specification`
3. **WebSearch**: `"<API name>" graphql schema site:github.com`
4. **WebSearch**: `"<API name>" API documentation developer reference`
5. Try common URL patterns for the API docs landing page
6. If a spec URL is found, **WebFetch** first 500 bytes to determine type:
   - Starts with `{"openapi":` or `openapi:` -> OpenAPI/REST
   - Contains `type Query {` or `schema {` -> GraphQL SDL
   - Contains `"__schema"` -> GraphQL introspection result

**Record the API type** (REST, GraphQL, or hybrid) for Phase 2's type check.

If no spec found: plan to write one from docs in Phase 2. For GraphQL APIs, the developer docs or a schema URL fetched via introspection serves the same role as an OpenAPI spec - it defines entities, fields, types, and relationships.

**Never refuse to proceed because you can't find a spec.** Write one from docs, or use GraphQL mode.

### Step 1.2: Search for competing CLIs

**WebSearch**: `"<API name>" CLI tool github`
**WebSearch**: `"<API name>" command line client`

Also search for non-wrapper tools discovered in Phase 0:
**WebSearch**: `"<API name>" sync OR archive OR export site:github.com`

For each competitor found, note repo URL, star count, language.

### Step 1.3: Deep competitor analysis (TOP 2 competitors)

For the top 2 competitors by stars, do ALL of the following:

1. **WebFetch** their README - count commands, note features, assess quality
2. **WebFetch** their GitHub repo main page - check:
   - Last commit date (is it maintained?)
   - Open issue count
   - Number of contributors
3. **WebSearch**: `site:github.com/<org>/<repo>/issues` - look for:
   - User complaints about missing features
   - Requests for specific functionality
   - Pain points users report
4. Record at least 2 specific user quotes or pain points.

### Step 1.4: Check demand signals

**WebSearch**: `"<API name>" "need a CLI" OR "command line" site:reddit.com OR site:news.ycombinator.com`

### Step 1.5: Strategic justification

Answer this question explicitly: **"Why should this CLI exist when [best competitor] already has [N] stars?"**

The answer must be SPECIFIC. Not just "agent-native." Examples:
- "[Competitor] hasn't been updated since [date] and doesn't support the latest API version"
- "No existing CLI supports --json + --select + --dry-run for agent workflows"
- "Users on [platform] are asking for [specific feature] which no CLI provides"
- "[Competitor] has [N] open issues about [problem] we can solve"

### Step 1.6: Write the research artifact

**Write** to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-research.md`:

```markdown
---
title: "Research: <API> CLI"
type: feat
status: active
date: <today>
---

# Research: <API> CLI

## Spec Discovery
- Official OpenAPI spec: <url or "none found - will write from docs">
- Source: <where found>
- Format: <OpenAPI 3.x / Swagger 2.0 / internal YAML>
- Endpoint count: <N>

## Competitors (Deep Analysis)

### <Competitor 1> (<stars> stars)
- Repo: <url>
- Language: <lang>
- Commands: <count>
- Last commit: <date>
- Open issues: <count>
- Maintained: <yes/no>
- Notable features: <list>
- Weaknesses: <what users complain about>

### <Competitor 2> (<stars> stars)
- [same structure]

## User Pain Points
> "<quote from GitHub issue or Reddit>" - <source>
> "<quote>" - <source>

## Auth Method
- Type: <api_key / oauth2 / bearer_token>
- Env var convention: <what competitors use>

## Demand Signals
- <specific posts with URLs, or "none found">

## Strategic Justification
**Why this CLI should exist:** <specific answer, not just "agent-native">

## Target
- Command count: <N - match or beat best competitor>
- Key differentiator: <specific features we'll have that competitors don't>
- Quality bar: Quality Grade A (80+/100)
```

### PHASE GATE 1

**STOP.** Verify ALL of these before proceeding:
1. Research artifact exists with Spec Discovery section
2. At least 2 competitors analyzed with maintenance status
3. At least 2 user quotes or pain points documented
4. Strategic justification answers "why should this exist?"
5. Target command count is set

**Write Phase 1 Artifact:** Run the Artifact Writing plan generator with all Phase 1 research as input. Write to `~/cli-printing-press/docs/plans/<today>-feat-<api>-cli-research.md`. Include: spec discovery, deep competitor analysis with quotes, demand signals, strategic justification, target command count.

Tell the user: "Phase 1 complete: Found [spec/no spec], [N] competitors. Best: [name] ([stars] stars, [commands] commands, last commit [date]). Strategic angle: [1-sentence justification]. Proceeding to generation."

---

# PHASE 2: GENERATE

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

### Step 2.0: API Type Check

Before generating, verify the spec matches the API:

1. **If spec is OpenAPI/Swagger** -> proceed to REST generation (Step 2.1)

2. **If spec is a GraphQL schema or the API is GraphQL-only** -> GRAPHQL MODE:

   Tell the user: "This API is GraphQL-only. The generator produces REST scaffolding, so Phase 2 will create the project structure (go.mod, client, config, helpers) but skip endpoint generation. All commands will be hand-written in Phase 4 using a GraphQL client. Phases 0, 0.5, 0.7, 1, 3, 4, 4.5, and 5 run normally - the research, prediction engine, data layer, workflows, dogfood, and scoring are all API-type-agnostic."

   **For GraphQL APIs, Phase 2 produces scaffolding only:**
   - Create the project directory structure (`cmd/`, `internal/cli/`, `internal/client/`, `internal/config/`, `internal/store/`)
   - Generate `go.mod` with cobra + SQLite dependencies + a GraphQL client (`github.com/hasura/go-graphql-client` or `github.com/machinebox/graphql`)
   - Generate `root.go` with global flags (--json, --select, --dry-run, --stdin, --yes, --no-cache)
   - Generate `config.go` with auth handling (API key via env var)
   - Generate `client.go` with a GraphQL client wrapper (POST to the single endpoint with query/variables)
   - Generate `helpers.go`, `doctor.go`, `auth.go`, `store.go` from templates
   - Generate `main.go`
   - DO NOT generate per-endpoint command files (there are no REST endpoints)

   **The GraphQL client wrapper should look like:**
   ```go
   func (c *Client) Query(query string, variables map[string]any) (json.RawMessage, error) {
       body := map[string]any{"query": query, "variables": variables}
       return c.do("POST", "/graphql", nil, body)
   }
   ```

   **Then proceed to Phase 2 Gate.** Phase 4 will build all commands by hand using the GraphQL schema.

3. **If the spec describes REST endpoints but the API base URL contains `/graphql`** ->
   Warn: "The spec describes REST endpoints but the API appears to be GraphQL. Double-check which is authoritative. If the REST spec is valid, proceed with REST generation. If GraphQL is the real API, switch to GraphQL mode."

### Step 2.0b: Validate Module Path

Before running the generator, check the module path it will use:

1. Run: `git config user.name` - this becomes the org in the module path
2. If this doesn't match your actual GitHub username, note the correct one
3. The module path should be: `github.com/<correct-github-username>/<product-name-from-phase-0.8>`
4. After generation, verify: `head -1 <api>-pp-cli/go.mod`
5. If wrong, fix with:
   ```bash
   cd <api>-pp-cli
   go mod edit -module github.com/<org>/<name>
   find . -name '*.go' -exec sed -i '' "s|<old-module>|<new-module>|g" {} +
   go build ./...
   ```

### Step 2.1: Get the spec ready

**If OpenAPI spec found:**
```bash
curl -sL -o /tmp/printing-press-spec-<api>.json "<spec-url>" && head -c 200 /tmp/printing-press-spec-<api>.json
```

**If no spec (write from docs):**
1. **WebFetch** the API docs
2. **Read** `~/cli-printing-press/skills/printing-press/references/spec-format.md`
3. Write YAML spec to `/tmp/<api>-spec.yaml` with ALL endpoints
4. Include auth config matching research findings

### Step 2.2: Check for existing output, remove if exists

```bash
cd ~/cli-printing-press && rm -rf library/<api>-cli 2>/dev/null; echo "CLEAN"
```

### Step 2.3: Run the generator

```bash
cd ~/cli-printing-press && ./printing-press generate \
  --spec /tmp/printing-press-spec-<api>.json \
  --output ./library/<api>-cli \
  --force --lenient --validate 2>&1
```

### Step 2.4: Note skipped complex body fields

**IMPORTANT:** When the generator outputs "warning: skipping body field X: complex type not supported", note EVERY skipped field. You will handle these in Phase 4.

Run:
```bash
cd ~/cli-printing-press && ./printing-press generate --spec /tmp/printing-press-spec-<api>.json --output ./library/<api>-cli --force --lenient --validate 2>&1 | grep "skipping body field"
```

Save the list of skipped fields. These are NOT acceptable limitations - they are work items for Phase 4.

### Step 2.5: Handle quality gate failures

Max 3 retries. Read errors carefully and fix spec issues.

### Step 2.7: Validate API Version Header

The generated client may pin to an outdated API version. Fix it:

1. Check what version the API docs say to use (search the API's changelog or docs)
2. Check the generated client's version header:
   ```bash
   grep -n "Version" <api>-pp-cli/internal/client/client.go
   ```
3. If the API uses date-based version headers (like Notion, Stripe), use the LATEST documented version, not the spec's version field or the generator's template default.
4. **Check for per-endpoint versioning.** Some APIs (e.g., Cal.com) use different version headers per resource — bookings may require `2024-08-13` while event-types requires `2024-06-14`. Test at least 2 different resource endpoints with the same version header. If one returns 404 or errors while the other succeeds, the API uses per-endpoint versioning. Implement routing logic in client.go:
   ```go
   apiVersion := "2024-06-14" // default
   if strings.Contains(path, "/bookings") || strings.Contains(path, "/slots") {
       apiVersion = "2024-08-13"
   }
   req.Header.Set("cal-api-version", apiVersion)
   ```
5. Update the header in client.go.
6. **Test with the live API** (if token available from Phase 0.1) to confirm the version header is accepted on at least 2 different resource types.

**Anti-shortcut:** "The generator's default is fine" - NO. Check it. "One version header works for all endpoints" — maybe not. Test two different resources.

### Step 2.8: Validate Config Env Var Matches Phase 0.1

The generator creates a config.go that looks for a specific env var name (e.g., `CAL_COM_USER_TOKEN`). This may not match the env var detected in Phase 0.1 (e.g., `CAL_COM_API_KEY`). If they don't match, live testing will fail with "auth: not configured" even though the user has the key set.

1. Check what env var name(s) Phase 0.1 detected or the user provided
2. Check what env var name config.go looks for: `grep "Getenv" <api>-pp-cli/internal/config/config.go`
3. If they differ: patch config.go to accept both names (check the common one first)
4. Also add any well-known env var names for this API (e.g., for Cal.com: `CAL_COM_API_KEY`, `CAL_API_KEY`, `CALCOM_API_KEY`)

### Step 2.9: Smoke Test (if API key available)

If an API key was provided in Phase 0.1, run a quick smoke test NOW — don't wait until Phase 5.5. This catches auth and version header issues before investing in Phase 3-4.

1. Build the CLI: `go build -o ./<product-name> ./cmd/<product-name>`
2. Run: `<product-name> doctor --json` — verify auth shows "configured"
3. Run: `<product-name> me <get-subcommand> --json` or equivalent profile endpoint — verify HTTP 200
4. Run one list endpoint for a different resource — verify no 404 (catches per-endpoint versioning issues)
5. If any test fails: fix the issue (version header, env var, base URL) before proceeding

This takes 30 seconds and can save hours of debugging in Phase 4.

### PHASE GATE 2

**STOP.** Verify:
1. CLI directory exists
2. `go build ./...` succeeds
3. List of skipped complex body fields is saved for Phase 3
4. Module path is correct (Step 2.0b)
5. API version header is current (Step 2.7)
6. Config env var matches Phase 0.1 detection (Step 2.8)
7. Smoke test passes if API key available (Step 2.9)

Tell the user: "Phase 2 complete: Generated <api>-pp-cli with [N] resources, [M] endpoints. [K] complex body fields noted for Phase 4. Proceeding to Non-Obvious Insight Review."

---

# PHASE 3: NON-OBVIOUS INSIGHT REVIEW

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

This phase has TWO parts: (A) code review for tactical fixes, and (B) Non-Obvious Insight Review for strategic assessment. Both are required.

## Part A: Code Review

### Step 3.1: Read the generated code

You MUST **Read** these files (not just check they exist):

- `library/<api>-cli/internal/cli/root.go`
- `library/<api>-cli/README.md`
- At least 3 resource command files

### Step 3.2: Count commands and compare to target

```bash
cd ~/cli-printing-press/library/<api>-cli && grep -r "Use:" internal/cli/*.go | grep -v "root.go" | wc -l
```

Compare against target from Phase 1 research.

### Step 3.3: Check help text quality

- Descriptions: developer-friendly or raw spec jargon?
- Examples: realistic values or placeholder garbage ("string", "0", "abc123")?
- Root command: does it explain what the API does?

### Step 3.4: Check for missing endpoints

Compare spec endpoints against generated commands. Note any gaps.

### Step 3.5: Check agent-native features

Verify in root.go: --json, --select, --dry-run, --stdin, --yes, --no-cache, doctor.

### Step 3.6: Check complex body fields

For each field skipped by the generator (from Phase 2 Step 2.4):
1. Is this field critical for the endpoint's purpose?
2. Can the user work around it with `--stdin`?
3. What example JSON would a user pipe in?

## Part B: Non-Obvious Insight Review

### Step 3.0: Run automated scorecard

Before hand-scoring, run the automated scorecard to get objective baseline numbers:

```bash
cd ~/cli-printing-press && ./printing-press scorecard --dir ./library/<api>-cli
```

Use these numbers as the baseline. The hand-scoring in Step 3.7 should explain WHY each dimension got its score, not re-guess the number.

### Step 3.7: Score against the quality bar

Score each dimension 0-10. For EACH dimension, provide THREE things:
1. **Current score** with justification
2. **What 10/10 looks like** (reference gogcli or best-in-class)
3. **What specific changes would raise the score** (actionable items)

```markdown
## Quality Assessment (Baseline)

| Dimension | Score | What 10 Looks Like | How to Get There |
|-----------|-------|-------------------|-----------------|
| Output modes | X/10 | gogcli: --json, --yaml, --csv, --table, --select, --quiet, --template | Add --yaml output, add --template for custom formats |
| Auth | X/10 | gogcli: OAuth browser flow, token storage, multiple profiles, doctor validates | Add OAuth flow, add profile switching |
| Error handling | X/10 | gogcli: typed exits, retry with backoff, helpful suggestions, link to docs | Add "did you mean?" suggestions |
| Terminal UX | X/10 | gogcli: progress spinners, color themes, pager for long output | Add progress spinners for pagination |
| README | X/10 | gogcli: install, quickstart, every command with example, cookbook, FAQ | Add cookbook section, add FAQ |
| Doctor | X/10 | gogcli: validates auth, API version, rate limits, config file health | Add API version check, config health |
| Agent-native | X/10 | gogcli: --json, --select, --dry-run, --stdin, idempotent, typed exits, no TTY | Already strong if all flags present |
| Local Cache | X/10 | gogcli: file cache + optional embedded DB (bolt/badger), --no-cache bypass, cache clear | [what changes would raise score] |
| Breadth | X/10 | gogcli: 100+ commands covering every API endpoint + convenience wrappers | Add missing commands, add convenience wrappers |
| Vision | X/10 | discrawl: SQLite + FTS5 + sync + search + tail + domain workflows | Add export, search, sync commands based on Phase 0 research |

**Baseline Total: X/100 (Grade X)**
```

### Step 3.7b: Head-to-Head Competitor Feature Matrix

The 10-dimension score (above) measures code quality against gogcli. This step measures feature completeness against the ACTUAL competition.

For the top 2 competitors (from Phase 1):

| Feature | Competitor A | Competitor B | Ours | Status |
|---------|-------------|-------------|------|--------|
| (list every command they have) | Y/N | Y/N | Y/N | HAVE / MISSING / BETTER |

MISSING features with >50% user need become Phase 4 Priority 1 work items.
BETTER features are our differentiators - highlight in README comparison table.

**The gogcli score tells you if the code is good.**
**The competitor matrix tells you if the product is good.**
**You need both.** A CLI that scores 10/10 on output modes but can't do what the top competitor does is a well-polished toy.

### Step 3.8: Write the GOAT improvement plan

Based on the quality analysis AND the competitor feature matrix, identify:

1. **TABLE STAKES gaps** (from Phase 0.6 + Step 3.7b - highest priority)
2. **Top 5 highest-impact quality improvements** (will raise the score the most)
3. **Commands to ADD** (not just rename - new functionality)
4. **Complex body field examples** to add (top 3 endpoints where --stdin matters most)
5. **What's achievable in Phase 4** vs what's future work

### Step 3.9: Write the audit artifact

**Write** to `~/cli-printing-press/docs/plans/<today>-fix-<api>-cli-audit.md`:

Include ALL of:
- Command comparison
- Help text quality assessment
- Agent-native checklist
- Specific fixes needed (file paths + what to change)
- Quality Assessment table (full)
- GOAT improvement plan (top 5 + commands to add)
- Complex body field plan

### PHASE GATE 3

**STOP.** Verify ALL of these:
1. Audit artifact exists with Quality Assessment table
2. Each quality dimension has: score, "what 10 looks like", and "how to get there"
3. GOAT plan has at least 5 specific improvements
4. Complex body fields have a plan (not just "limitation")
5. Baseline total score is recorded

**Write Phase 3 Artifact:** Run the Artifact Writing plan generator with all Phase 3 analysis as input. Write to `~/cli-printing-press/docs/plans/<today>-fix-<api>-cli-audit.md`. Include: scorecard baseline, full 11-dimension hand-scored table, GOAT improvement plan, complex body field plan, data layer integration notes from Phase 0.7.

Tell the user: "Phase 3 complete: Baseline Quality Score: [X]/100 (Grade [X]). Found [N] tactical fixes + [M] GOAT improvements. Top improvement: [description]. Proceeding to GOAT build."

---

# PHASE 4: GOAT BUILD

## THIS PHASE IS MANDATORY.

**The generator output is scaffolding, not the product. The data layer + workflows are the product.**

**Read the Phase 0.7 Data Layer Specification and Phase 3 Audit artifacts before starting.**

**GraphQL APIs:** For GraphQL APIs, Phase 2 only produced scaffolding (no generated commands). Phase 4 is where ALL commands get written by hand. Use the GraphQL schema + Phase 0.5 workflows + Phase 0.7 data layer spec to determine which queries/mutations to wrap as CLI commands. Each command sends a GraphQL query via the client wrapper. Prioritize workflow commands over CRUD wrappers - a `linear-cli stale --days 30 --team ENG` is more valuable than `linear-cli issues list`.

Execute in this priority order. Do NOT skip Priority 0 to go straight to workflows.

### Codex Delegation in Phase 4

If CODEX_MODE is true, each Priority 0/1/2/3 task below is a **separate Codex call**:
- Claude reads the Phase 0.7 spec and Phase 3 audit to decide WHAT to build
- Claude assembles a Codex prompt for each task with: the specific file, current code context, expected behavior
- Codex writes the code (one task = one Codex call, scoped to 1-2 files, <200 lines)
- Claude reviews the diff, runs `go build ./... && go vet ./...`
- If Codex fails: Claude writes that task directly (fallback)
- After all tasks: run Proof of Behavior verification to catch any issues

**Example Codex prompt for store.go rewrite:**
```
TASK: Rewrite store.go with domain-specific tables for Discord.

FILES TO MODIFY:
- discord-cli/internal/store/store.go

CURRENT CODE (Open function signature):
func Open(dbPath string) (*Store, error) { ... }

EXPECTED CHANGE:
Replace the generic resources table with domain-specific tables:
- messages: id, channel_id, guild_id, author_id, content, timestamp, data JSON
- members: guild_id, user_id, username, display_name, roles JSON, data JSON
- channels: id, guild_id, name, type, parent_id, data JSON
Add FTS5 on messages.content and members.username.
Add UpsertMessage, UpsertMember, UpsertChannel methods.
Add SearchMessages method using FTS5 MATCH.

CONVENTIONS:
- Use modernc.org/sqlite (pure Go, no CGO)
- WAL mode + synchronous=NORMAL + mmap_size=268435456
- FTS5 with content='table', content_rowid='rowid'

CONSTRAINTS:
- Do NOT run git commit/push/add
- Keep under 400 lines (store.go can be longer than typical)
- Run: cd discord-cli && go build ./... && go vet ./...
```

### Priority 0: Data Layer Foundation (from Phase 0.7)

**This is the most important work in the entire pipeline.** Replace the generic store with domain-specific tables.

1. **Replace `internal/store/store.go`** with domain-specific schema from Phase 0.7 spec:
   - CREATE TABLE with proper columns for each Primary entity (not JSON blobs)
   - CREATE INDEX on foreign keys and temporal fields
   - CREATE VIRTUAL TABLE ... USING fts5() on text fields
   - UpsertMessage/UpsertMember/etc methods that extract fields from JSON

2. **Rewrite `sync` command** with domain-aware sync:
   - Use the cursor field identified in Phase 0.7 (e.g., snowflake ID, updatedAt)
   - Add `--guild`/`--team`/`--project` scoping flags (domain-specific)
   - Add `--since` time filter
   - Paginate with the validated cursor params

3. **Add domain-specific search filters:**
   - Map Phase 0.7's filter table into search command flags
   - `--channel`, `--author`, `--team`, etc -> SQL WHERE clauses
   - FTS5 on extracted text content, not raw JSON

4. **Add `sql` command** for raw read-only queries:
   ```go
   func newSqlCmd(flags *rootFlags) *cobra.Command {
       cmd := &cobra.Command{
           Use:   "sql <query>",
           Short: "Run read-only SQL queries against the local database",
           RunE: func(cmd *cobra.Command, args []string) error {
               // Open DB, execute query, print results as table or JSON
           },
       }
   }
   ```

5. **Add entity-specific list commands** (e.g., `messages`, `members`):
   - Query local SQLite, not the API
   - Support `--channel`, `--author`, `--days`, `--hours`, `--since` filters
   - Support `--sync` flag to trigger on-demand sync before querying

6. **Update tail command** based on Phase 0.7's tail strategy:
   - If WebSocket/Gateway: implement real-time connection
   - If SSE: implement EventSource reader
   - If REST polling: keep current implementation but use domain-aware cursors

### Priority 1: Table Stakes Features (from Phase 0.6)

**Build every feature classified as TABLE STAKES in Phase 0.6.** These are features that the top competitor has and that >50% of users expect. This is NOT optional polish - these are the features that determine whether anyone would switch from the incumbent.

For each table-stakes feature:
1. Read how the competitor implements it (from Phase 1 research)
2. Implement it - don't just match the competitor, make it BETTER
3. Better means: works with --json, supports --dry-run, has --stdin, composes with our data layer where possible
4. Register in root.go alongside the generated commands

**Gate:** Every TABLE STAKES feature from Phase 0.6 must be implemented before proceeding to Priority 2. No exceptions.

### Priority 2: Power User Workflows (from Phase 0.5) - NOW powered by local DB

Implement the top 5-7 workflows identified in Phase 0.5 as real, hand-written Go commands. **Where possible, query the local SQLite database instead of making live API calls.** This makes workflows instant and avoids rate limits.

For each workflow:
1. **Create a dedicated command file** (e.g., `internal/cli/stale.go`, `internal/cli/velocity.go`)
2. **Use the generated client** to make real API calls
3. **Combine 2+ API calls** into one user-facing operation
4. **Add realistic examples** in --help that show the actual workflow
5. **Support --json output** for agent consumption
6. **Register in root.go** alongside the generated commands

### Priority 3: Command Name Normalization + Apply Product Name

**Step 3a: Normalize generated command names**

The generator produces ugly operationId-derived names. Fix them:

| Generated | Normalized | Rule |
|-----------|-----------|------|
| `retrieve-a*` | `get` | Strip "Retrieve a" prefix |
| `delete-a*` | `delete` | Strip "Delete a" prefix |
| `create-a*` | `create` | Strip "Create a" prefix |
| `update-a*` | `update` | Strip "Update a" prefix |
| `post` | `create` | HTTP method -> action |
| `patch` | `update` | HTTP method -> action |
| `get-self` | `me` | Special case |
| `list-*` | `list` | Strip resource suffix |

For each rename: update the `Use:` field, rename the file, verify `go build` passes.

**Step 3b: Apply the product name everywhere**

1. Rename `cmd/<generated-name>/` to `cmd/<product-name>/`
2. Update root.go Use field and version template
3. Update go.mod module path to `github.com/<org>/<product-name>`
4. Update client.go User-Agent header
5. `grep -r "<old-name>" . | grep -v "Generated by"` must return 0 hits — **pay special attention to DB path strings and config path strings** (e.g., `~/.local/share/<old-name>/data.db`, `~/.config/<old-name>/config.toml`). These are easy to miss and will cause the search/analytics/export commands to use a different database than the sync command. If the old name appears in any `filepath.Join` or `defaultDBPath` call, fix it.
6. Update README examples

**Step 3c: Validate API version header**

1. Check what API version the spec uses / the API docs recommend
2. Check what the generated client sends (`grep "Version" client.go`)
3. If they differ, update client.go to the latest documented version
4. If the API uses date-based versions (Notion, Stripe), use the LATEST

### Priority 4: Scorecard Gap Fixes (DEMOTED + ANTI-GAMING)

Run the scorecard. Fix REAL gaps. DO NOT GAME IT.

```bash
cd ~/cli-printing-press && ./printing-press scorecard --dir ./library/<api>-cli
```

**ANTI-GAMING RULES:**
- If a function exists only because the scorecard checks for a string pattern, DELETE IT.
- If a flag is registered but never checked in any RunE, DELETE IT.
- If an import exists only to put "store." in the file, DELETE IT.
- A CLI that scores 60 but has every table-stakes feature beats one that scores 80 with type aliases.
- The scorecard measures proxies for quality. Optimize for actual quality.

Also fix:
- Complex body field --stdin examples for top 3 endpoints
- Lazy descriptions (1-2 word Short fields)
- Placeholder examples ("abc123" -> realistic domain values)

### Priority 5: Write Tests

A CLI with 0 test files is not shippable.

For each Primary entity in the data layer:
1. Test UpsertX with valid data -> verify row in DB
2. Test UpsertX with missing fields -> verify graceful handling
3. Test SearchX with FTS5 -> verify results match

For each workflow command:
1. Seed DB with test fixtures
2. Run the command's core query
3. Verify result shape and counts

Minimum: 1 test file per package (store, cli). Use table-driven tests matching Go conventions.

### Priority 6: Distribution Scaffold

1. Add `.goreleaser.yaml` for cross-platform binary builds
2. Add Homebrew formula or tap
3. Add install instructions for non-Go users to README
4. Add `.github/workflows/ci.yml` (go test, go vet, goreleaser on tag)

A CLI that can only be installed via `go install` is not a real CLI.

### Priority 7: Polish

Only after all above priorities are complete:
1. README cookbook section **showcasing workflow commands AND table-stakes features** (not just API calls)
2. FAQ section with domain-specific questions
3. Comparison table in README showing where we beat each competitor

### Step 4.4: Verify compilation

```bash
cd ~/cli-printing-press/library/<api>-cli && go build ./... && go vet ./... && echo "ALL FIXES VERIFIED"
```

### PHASE GATE 4

**STOP.** Verify ALL priorities:

**Priority 0 (Data Layer):**
1. Domain-specific SQLite tables (NOT generic JSON blobs)
2. Sync command uses domain-aware cursors (validated in Phase 0.7)
3. Search supports domain-specific filters (--channel, --author, --team, etc.)
4. `sql` command exists for raw read-only queries

**Priority 1 (Table Stakes):**
5. Every TABLE STAKES feature from Phase 0.6 is implemented
6. Table stakes features work with --json, --dry-run, --stdin where applicable

**Priority 2 (Workflows):**
7. At least 3 workflow commands implemented (from Phase 0.5)
8. Workflow commands use local DB where possible (not just live API calls)

**Priority 3 (Names):**
9. Command names normalized (get/create/update/delete, not retrieve-a/post/patch)
10. Binary name matches product name from Phase 0.8
11. Module path is correct

**Priority 4-7:**
12. `go build ./...` and `go vet ./...` pass
13. At least 1 test file per package (store, cli)
14. README cookbook includes data layer + workflow + table-stakes examples

**Data Pipeline Trace (MANDATORY):** For each Primary entity from Phase 0.7, verify:
- WRITE path exists: sync.go calls `db.UpsertX()` for this entity (file:line)
- READ path exists: at least one command queries this entity's table (file:line)
- SEARCH path exists (if FTS5): at least one command calls `db.SearchX()` (file:line)
- If ANY Primary entity has no WRITE path, the data layer is broken. Fix before proceeding.

**Three-Benchmark Check (MANDATORY):**

1. ARCHITECTURE (discrawl benchmark): "Does this CLI have a real data layer - domain-specific SQLite tables, FTS5 search, incremental sync, workflow commands that query local data?" If no, Priority 0 isn't done.

2. QUALITY (gogcli benchmark): "Does the code have proper output modes, typed errors, agent-native flags, doctor command, README with cookbook?" If gaps, Priority 4 scorecard fixes address them.

3. FEATURES (competitor benchmark): "Would a user of [top competitor from Phase 1] switch to this CLI?" If no: "What's the ONE feature that would flip them?" Build it now before proceeding.

All three must pass. Architecture without features is a toy. Features without architecture is a thin wrapper. Quality without either is polished nothing.

**Write Phase 4 Artifact:** Run the Artifact Writing plan generator with all Phase 4 work as input. Write to `~/cli-printing-press/docs/plans/<today>-fix-<api>-cli-goat-build-log.md`. Include: data layer implementation details, workflow commands built, scorecard fixes, what was skipped, before/after scorecard comparison.

Tell the user: "Phase 4 complete: Built [N] data layer tables + [M] workflow commands, applied [K] scorecard fixes. Data layer: [list tables]. Top workflow: [name]. Compilation verified. Proceeding to dogfood emulation."

---

# PHASE 4.5: DOGFOOD EMULATION

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

You don't have real API keys. But the OpenAPI spec already defines every request shape, response schema, error format, and pagination pattern. Test every generated command against spec-derived mock responses. Inspired by [Vercel's emulate](https://github.com/vercel-labs/emulate) - production-fidelity API simulation, zero config.

**Read the OpenAPI spec (from Phase 2) and the Phase 4 GOAT Build Log artifact before starting.**

### Step 4.5a: Generate Synthetic Responses from Spec

For each endpoint in the spec, generate a realistic JSON response by reading the 200/201 response schema:

**Field value heuristics** (domain-aware, not random):

| Field name pattern | Generated value |
|---|---|
| `id`, `*_id` | Realistic format for the API (e.g., Discord snowflake: `"1234567890123456789"`, UUID: `"550e8400-e29b-41d4-a716-446655440000"`) |
| `name`, `username`, `title` | Realistic domain values (e.g., `"general"`, `"test-user"`, `"Bug: Login fails"`) |
| `content`, `description`, `body` | `"Dogfood test content for validation"` |
| `timestamp`, `created_at`, `updated_at` | `"2026-03-26T12:00:00.000Z"` |
| `type` (enum in spec) | First enum value from the spec |
| `url`, `avatar_url`, `icon_url` | `"https://example.com/test.png"` |
| `email` | `"test@example.com"` |
| `count`, `position`, `size` | `1` |
| `boolean` fields | `true` |
| Array fields | 2-3 items with the above heuristics |
| Nested objects | Recursively generate from schema |

Save mocks to `/tmp/<api>-cli-mocks/` for reuse.

### Step 4.5b: Score Every Command on 5 Dimensions

For each generated command, score 0-10 on each dimension (50 max):

**Dimension 1: Request Construction (0-10)**

Run the command with `--dry-run` and inspect the output:

```bash
<api>-cli <resource> <action> <required-args> --dry-run 2>&1
```

| Check | Points |
|---|---|
| Path params replaced (no `{param}` literals in URL) | 2 |
| HTTP method matches spec | 2 |
| Required query params present | 2 |
| Body schema matches spec's requestBody | 2 |
| Auth header present | 2 |

**Dimension 2: Response Parsing (0-10)**

Generate a synthetic response from the spec and verify the command can process it:

| Check | Points |
|---|---|
| Can parse the spec's 200 response schema | 3 |
| --json output is valid JSON | 2 |
| --select works with fields from the response schema | 2 |
| Table output renders without crash | 2 |
| Error responses (401/404/429) produce correct exit codes | 1 |

**Dimension 3: Schema Fidelity (0-10)**

Compare generated flags against the spec's parameters:

| Check | Points |
|---|---|
| All `required: true` params have CLI flags | 3 |
| Flag types match spec types (string/int/bool) | 2 |
| No hallucinated flags (every flag maps to a real spec param) | 3 |
| Help descriptions come from spec, not invented | 2 |

**Dimension 4: Example Quality (0-10)**

Validate every example in --help and README:

| Check | Points |
|---|---|
| Example IDs match realistic format (not "abc123") | 2 |
| --stdin JSON matches spec's requestBody schema | 3 |
| Required flags present in examples | 3 |
| Example commands parse without usage error via --dry-run | 2 |

**Dimension 5: Workflow Integrity (0-10)** (workflow commands only)

| Check | Points |
|---|---|
| All API paths hit by the workflow exist in the spec | 3 |
| Query params sent match spec's parameters | 2 |
| Response fields accessed exist in the response schema | 3 |
| Cross-entity joins reference valid fields | 2 |

### Step 4.5c: Compute Aggregate Scores

| Metric | Formula |
|---|---|
| **Per-command score** | Sum of 5 dimensions (0-50) |
| **Pass rate** | % of commands scoring >= 35/50 (70%) |
| **Critical failure count** | Commands scoring < 25/50 |
| **Overall dogfood score** | Average across all tested commands |

**Thresholds:**
- **PASS:** Pass rate >= 90% AND 0 critical failures
- **WARN:** Pass rate >= 70% AND <= 3 critical failures (auto-fix, then re-score)
- **FAIL:** Pass rate < 70% OR > 3 critical failures (report issues, do NOT proceed)

**Sampling for large CLIs (100+ commands):** Test ALL workflow commands + ALL commands with --stdin examples + random sample of 30 generated commands. Report sample size.

### Step 4.5d: Write the Dogfood Report ("Here's what I learned")

**Run the Artifact Writing plan generator** with all dogfood scoring results as input. Write to `~/cli-printing-press/docs/plans/<today>-fix-<api>-cli-dogfood-report.md`.

The report MUST include three sections:

**Section 1: "Here's what I learned"**
- Per-command score table (sampled commands, all 5 dimensions)
- Top 5 failures with root cause analysis
- Hallucination list (flags/fields not in spec with evidence)
- Pattern analysis: what categories of issues keep appearing?
- Comparison: workflow commands vs generated commands - which score better?

**Section 2: "Here's what I think we should fix"**
- Prioritized fix list, ordered by impact (critical failures first)
- For each fix: what's wrong, which file, what the fix is, expected score improvement
- Mark each as AUTO-FIXABLE or NEEDS-MANUAL-FIX
- Estimate: "fixing these [N] issues would raise pass rate from [X]% to [Y]%"

**Section 3: "Here's what I think we should make"**
- Features or commands that the dogfood revealed are missing
- Endpoints that exist in the spec but have no generated command
- Schema fields that should be in the data layer but aren't
- Workflow ideas that emerged from understanding the API responses better

### Step 4.5e: Fix Everything Fixable

Now execute the fixes from the report. For each AUTO-FIXABLE issue:

| Issue Type | Fix Action |
|---|---|
| Placeholder values ("abc123", "string") | Replace with realistic domain values from spec |
| Missing required flags in examples | Add required flags with domain-realistic values |
| --stdin JSON doesn't match requestBody | Regenerate from spec schema |
| Lazy 1-word Short descriptions | Pull description from spec's endpoint summary |
| Hallucinated flag (not in spec) | Remove the flag and its binding |
| Wrong flag type (string instead of int) | Fix the cobra flag type |
| Path param not substituted in example | Fix the example with a realistic ID |
| Workflow hits nonexistent endpoint | Fix the path or remove the workflow |

**After ALL fixes:**
1. Run `go build ./...` and `go vet ./...` to verify fixes compile
2. Re-run the FULL dogfood scoring (Steps 4.5b + 4.5c) on the same sample
3. Compute the delta: before/after per-dimension and aggregate scores
4. **Update the dogfood report artifact** with a new section:

**Section 4: "Here's what we fixed" (appended after Step 4.5e)**
- List of every fix applied with file path and description
- Before/after scores per command that was fixed
- Aggregate improvement: pass rate delta, critical failure delta, avg score delta
- Remaining unfixed issues (NEEDS-MANUAL-FIX items) with reasons

### Step 4.5f: Implement "Should Make" Recommendations (if time permits)

Review the "Here's what I think we should make" section from the report. For each recommendation:

1. Is this a quick win (< 10 min to implement)? -> Do it now
2. Is this a significant feature (> 10 min)? -> Add to the dogfood report as "Future Work"
3. Does this improve the quality score? -> Prioritize it

After implementing quick wins, re-run `go build` and the dogfood on affected commands.

### PHASE GATE 4.5

**STOP.** Verify ALL of these before proceeding:
1. Every workflow command scored on all 5 dimensions
2. Sample of generated commands scored (30+ or all if < 100)
3. Synthetic responses generated from spec (not invented)
4. Per-command score table computed
5. Dogfood report written with all 4 sections (learned, should fix, should make, fixed)
6. All AUTO-FIXABLE issues resolved
7. Re-score after fixes shows measurable improvement
8. `go build ./...` and `go vet ./...` pass
9. Final verdict: PASS or WARN (FAIL = stop and report, do NOT proceed)

Tell the user: "Phase 4.5 complete: Dogfood score [X]/50 avg across [N] commands (was [X0] before fixes, +[D] improvement). Pass rate: [Y]% (was [Y0]%). Critical failures: [Z] (was [Z0]). Auto-fixed [K] issues. Implemented [J] quick-win recommendations. [PASS/WARN/FAIL]. Proceeding to hallucination audit."

---

# PHASE 4.6: HALLUCINATION & DEAD CODE AUDIT

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

The scorecard tests syntax. This phase tests semantics. Every flag, function, and table must be wired to real code paths. Dead code that exists only to trigger scorecard string matches is worse than missing code.

### Step 4.6a: Dead Flag Audit

For every flag registered in root.go's persistent flags (--json, --csv, --stdin, --quiet, --yes, etc.):
1. Grep all RunE functions across internal/cli/*.go for `flags.<fieldName>`
2. If a flag is declared but never checked in any RunE: it's a **dead flag**
3. **FIX:** Either wire the flag into at least one command's RunE logic, OR remove the flag from root.go

### Step 4.6b: Dead Function Audit

For every function defined in helpers.go:
1. Grep all other .go files in internal/cli/ for that function name
2. If a function is defined but never called: it's a **dead function**
3. **FIX:** Either call the function from a real command's code path, OR delete the function

**WARNING:** Do NOT fix dead functions by adding calls that don't do anything useful. `_ = filterFields(nil)` in an init() block is gaming. The function must be called in a real output or error path.

### Step 4.6c: Ghost Table Audit

For every CREATE TABLE in store.go's migration:
1. Grep sync.go for an INSERT or Upsert call targeting this table
2. Grep all command files for a SELECT targeting this table
3. If a table has no INSERT path: it's a **ghost table** - created but never populated
4. **FIX:** Wire sync to populate the table via the domain-specific Upsert method, OR remove the table

### Step 4.6d: Data Pipeline Trace

For each Primary entity from Phase 0.7, trace the complete data flow:

| Entity | WRITE path (sync -> UpsertX) | READ path (command -> SELECT) | SEARCH path (command -> SearchX) |
|--------|------------------------------|-------------------------------|----------------------------------|

Every Primary entity MUST have a WRITE path. If any Primary entity has no write path, the data layer is broken. Fix before proceeding.

### PHASE GATE 4.6

**STOP.** Verify:
1. Dead flags: 0
2. Dead functions: 0
3. Ghost tables: 0
4. Every Primary entity has WRITE + READ paths
5. Present the data pipeline trace table to the user

Tell the user: "Phase 4.6 complete: [N] dead flags fixed, [M] dead functions removed, [K] ghost tables wired. Data pipeline verified for [X] primary entities."

---

# PHASE 4.8: RUNTIME VERIFICATION

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

The scorecard measures files. This phase measures behavior. Build the CLI and test every command.

### Step 4.8a: Run the Runtime Verifier

```bash
cd ~/cli-printing-press && ./printing-press verify \
  --dir ./library/<api>-cli \
  --spec /tmp/<api>-spec.json \
  --threshold 80
```

If you collected an API key in Phase 0.1, add it:

```bash
cd ~/cli-printing-press && ./printing-press verify \
  --dir ./library/<api>-cli \
  --spec /tmp/<api>-spec.json \
  --api-key "$<API_ENV_VAR>" \
  --env-var <API_ENV_VAR> \
  --threshold 80
```

The verifier:
1. Builds the CLI binary
2. Starts a mock server (or uses the real API if key provided - read-only GETs only)
3. Tests every discovered command: --help, --dry-run, --json execution
4. Tests the data pipeline end-to-end: sync -> sql -> search -> health
5. Scores each command (0-3) and computes aggregate pass rate

### Step 4.8b: Interpret Results

- **PASS** (>= 80% pass rate, data pipeline works, 0 critical): Proceed to Phase 5.
- **WARN** (60-80%): Review failures. Fix the top 3 manually and re-run.
- **FAIL** (< 60% or data pipeline broken): DO NOT proceed. Fix until at least WARN.

For each failing command, the verifier reports which test failed (help/dry-run/execute). Fix the root cause:
- Help fails = command not registered in root.go
- Dry-run fails = dryRun flag not checked in RunE
- Execute fails = wrong path, bad response parsing, or missing required flags

### PHASE GATE 4.8

**STOP.** Verify:
1. `printing-press verify` ran to completion
2. Pass rate >= 80%
3. Data pipeline: sync populates tables, sql queries them, search finds results
4. 0 critical failures

Tell the user: "Runtime verification: [X]% pass rate ([N]/[M] commands). Data pipeline: [PASS/FAIL]. Mode: [live/mock]. Proceeding to agent readiness review."

---

# PHASE 4.9: AGENT READINESS REVIEW LOOP

## THIS PHASE IS MANDATORY. YOU MUST ATTEMPT IT.

**You MUST dispatch the `compound-engineering:cli-agent-readiness-reviewer` agent.** Do not skip this phase because "it's faster to move on" or "the CLI is already good." The user wants this review to run when the agent is available.

**The only acceptable outcomes are:**
1. The agent ran and produced results (proceed with fixes)
2. The agent dispatch failed because the plugin is not installed (warn prominently, proceed to Phase 5)
3. The agent ran but produced an error or empty output (warn prominently, proceed to Phase 5)

**"I decided to skip it" is NOT an acceptable outcome.** If you catch yourself about to move to Phase 5 without having attempted the agent dispatch, STOP and go back.

The existing agent-native scorecard dimension checks for flags. This phase goes deeper — evaluating 7 principles (non-interactive automation, structured output, progressive help, actionable errors, safe retries, composability, bounded responses) with file-level fix recommendations.

### Step 4.9a: Dispatch the Agent (MANDATORY — FOREGROUND ONLY)

You MUST execute this Agent tool call **in the foreground** (NOT background). You need the results before you can proceed — do NOT use `run_in_background: true`.

```
Agent tool:
  subagent_type: compound-engineering:review:cli-agent-readiness-reviewer
  run_in_background: false
  prompt: "Run the compound-engineering:cli-agent-readiness-reviewer agent on the <api> CLI in <output-dir>.
           Do not look at code elsewhere in the repo outside of that folder."
```

**WAIT for the agent to return results before moving on.** Do NOT proceed to Phase 5 while this is running.

**If the dispatch succeeds:** proceed to Step 4.9b with the results.

**If the dispatch fails** (agent not found, plugin not installed, tool rejected):
1. Print this warning to the user in a visible block:

```
⚠️  PHASE 4.9 SKIPPED — REVIEWER NOT AVAILABLE
    The Compound Engineering `cli-agent-readiness-reviewer` agent could not be dispatched.
    This reviewer evaluates whether the generated CLI is usable by AI agents
    (exit codes, output routing, alias correctness, delete safety gates, etc.).
    Reason: [error message]
    To enable: install the compound-engineering plugin (v2.55.0+)
    and register every-marketplace in ~/.claude/settings.json.
    The CLI was NOT reviewed for agent readiness.
```

2. Proceed to Phase 5. Do NOT silently continue.

### Step 4.9b: Process Initial Results

**Set `pass_count = 0`.** The initial dispatch establishes the baseline — pass counting begins in Step 4.9d when you re-dispatch after fixes.

The reviewer produces:
- A scorecard table (7 principles x severity: Blocker/Friction/Optimization/None)
- A "What's Working Well" section
- A ranked list of recommended fixes with file:line references

**If the reviewer returned results but they are empty or unparseable:**
1. Print this warning to the user:

```
⚠️  PHASE 4.9 INCOMPLETE — REVIEWER RETURNED NO ACTIONABLE RESULTS
    The Compound Engineering `cli-agent-readiness-reviewer` agent ran but produced
    no parseable fix list. The CLI was NOT fully reviewed for agent readiness
    (exit codes, output routing, alias correctness, delete safety gates, etc.).
```

2. Proceed to Phase 5. Do not loop.

**If the reviewer returned a valid scorecard and fix list:** proceed to Step 4.9c.

### Step 4.9c: Implement Fixes

For each fix in the reviewer's ranked recommended fixes list:

1. Read the fix description and target file:line
2. If the referenced file does not exist or the line is out of bounds: skip this fix, log a warning
3. If `CODEX_MODE` is true, treat this fix as a **separate Codex call** using the Codex Delegation Pattern from the top of this skill:
   - Claude assembles a prompt with the exact reviewer finding, exact file:line, current code snippet, expected behavioral change, repo conventions, and the same constraints (`no git`, listed files only, `go build ./... && go vet ./...` at the end)
   - Scope the call to the smallest possible patch, ideally 1 file and under 100 lines
   - After Codex returns, immediately verify the target file still exists and is non-empty before reviewing the diff
   - If Codex fails, empties/deletes the file, edits outside scope, or fails verification: revert that attempted change and apply the fix directly in Claude as a fallback
4. If `CODEX_MODE` is false, make the code change directly in Claude
5. Run `go build ./... && go vet ./...` to verify the change compiles and passes vet
6. If build or vet fails: revert the change, skip this fix, continue to next
7. Move to the next fix

All listed fixes are attempted — the reviewer already ranks by impact.

**After all fixes are attempted, proceed to Step 4.9d. Do NOT skip to Phase 5 or the Phase Gate.** You must re-dispatch the reviewer to verify fixes before evaluating termination.

**Important:** The reviewer agent itself always runs inside Claude Code. Only the code-writing step is delegated. This preserves the same split as Phase 4: Claude reviews, prioritizes, and verifies; Codex writes the patch when the run started in Codex mode.

### Step 4.9d: Re-dispatch Reviewer and Termination Check (MANDATORY)

**You MUST re-dispatch the reviewer agent after implementing fixes.** Do NOT evaluate termination based on your own assessment of the fixes — only the reviewer's new scorecard counts. This re-dispatch is mandatory regardless of how confident you are that fixes resolved the issues.

**If you are about to proceed to Phase 5 without re-dispatching the reviewer after Step 4.9c, STOP — you are skipping a mandatory step. Go back and dispatch the agent.**

Increment `pass_count` by 1, then re-run the `compound-engineering:cli-agent-readiness-reviewer` agent on the same folder (same foreground dispatch as Step 4.9a — do NOT background). **WAIT for results.**

Evaluate the new scorecard:

- **Zero Blockers AND zero Frictions:** Pass. Proceed to Phase Gate 4.9.
- **Blockers or Frictions remain AND `pass_count < 2`:** Return to Step 4.9c with the new fix list. After implementing those fixes, return here — increment `pass_count` and re-dispatch again.
- **Blockers or Frictions remain AND `pass_count >= 2`:** Log remaining issues as known items. Proceed to Phase Gate 4.9.

**If the agent becomes unavailable between passes** (plugin timeout, tool error):
1. Print this warning to the user:

```
⚠️  PHASE 4.9 INTERRUPTED — REVIEWER LOST BETWEEN PASSES
    The Compound Engineering `cli-agent-readiness-reviewer` agent became unavailable
    after pass [N]. Fixes from pass [N] were applied. No further review possible.
```

2. Proceed to Phase Gate 4.9. The pass count does not reset.

### PHASE GATE 4.9

**STOP.** Evaluate the result:

| Verdict | Condition | Action |
|---------|-----------|--------|
| **Pass** | Zero Blockers and zero Frictions after ≤ 2 passes | Proceed to Phase 5 |
| **Warn** | Frictions remain after 2 passes, zero Blockers | Log Frictions as known issues, proceed to Phase 5 |
| **Degrade** | Blockers remain after 2 passes | Log Blockers as known issues, proceed to Phase 5 |
| **Skipped** | Agent unavailable (dispatch failed) | Warning already shown, proceed to Phase 5 |

Tell the user: "Agent readiness review: [PASS/WARN/DEGRADE/SKIPPED]. Blockers: [N]. Frictions: [N]. Optimizations: [N]. Passes: [N]/2. Proceeding to final report."

---

# PHASE 5: FINAL QUALITY SCORE + REPORT

## THIS PHASE IS MANDATORY. DO NOT SKIP IT.

### Step 5.1: Ship Readiness Assessment

Run the automated scorecard again to measure improvement:

```bash
cd ~/cli-printing-press && ./printing-press scorecard --dir ./library/<api>-cli
```

Re-score ALL 10 dimensions. Show the DELTA from the baseline:

```markdown
## Ship Readiness Assessment (Post-Fix)

| Dimension | Before | After | Delta | What Changed |
|-----------|--------|-------|-------|-------------|
| Output modes | X/10 | Y/10 | +Z | [specific change] |
| Auth | X/10 | Y/10 | +Z | [specific change] |
| Error handling | X/10 | Y/10 | +Z | [specific change] |
| Terminal UX | X/10 | Y/10 | +Z | [specific change] |
| README | X/10 | Y/10 | +Z | [specific change] |
| Doctor | X/10 | Y/10 | +Z | [specific change] |
| Agent-native | X/10 | Y/10 | +Z | [specific change] |
| Local Cache | X/10 | Y/10 | +Z | [specific change] |
| Breadth | X/10 | Y/10 | +Z | [specific change] |
| Vision | X/10 | Y/10 | +Z | [specific change] |

**Before: X/100 -> After: Y/100 (+Z points)**
**Grade: [A/B/C]**
```

### Step 5.2: Remaining gaps

For each dimension still < 8/10:
- What would it take to reach 8+?
- Is this a generator limitation or achievable with more fix time?
- Tag as "future work" with a specific next step

### Step 5.3: Present the final report

Show ALL of these sections:

**1. Summary:**
```
Generated <api>-cli with <N> resources and <M> commands.
Resources: <comma-separated list>
```

**2. Quality Score (Before/After):**
```
Quality Score: Before X/100 -> After Y/100 (+Z points) - Grade [A/B/C]

[Full before/after table from Step 5.1]
```

**3. Competitor Comparison:**
```
Found <N> competing CLIs.
Best competitor: <name> (<stars> stars, <commands> commands)
Strategic advantage: <why ours is better - from Phase 1 research>
We beat them on: <specific features>
Remaining gap: <what they have that we don't, or "none">
```

**4. Example Commands (with complex body examples):**
```bash
cd ~/cli-printing-press/library/<api>-cli
go install ./cmd/<api>-cli

export <AUTH_ENV_VAR>="..."

# Basic usage
<api>-cli --help
<api>-cli doctor
<api>-cli <resource> list --json
<api>-cli <resource> get <realistic-id>

# Complex body fields (pipe JSON via stdin)
echo '<realistic-json>' | <api>-cli <resource> create --stdin

# Agent workflow
<api>-cli <resource> list --json --select id,name | jq -r '.[].id'
```

**5. Spec source and limitations**

**6. Future work** (from remaining gaps)

---

# PHASE 5.5: LIVE API TESTING (optional - requires API key from Phase 0.1)

Skip this phase entirely if no API key was provided in Phase 0.1.

## Safety Rules (NON-NEGOTIABLE)

These rules CANNOT be overridden. Violation = immediate abort.

1. ONLY execute HTTP GET operations (list, get, search, doctor)
2. NEVER execute POST, PUT, PATCH, DELETE (no creating, updating, deleting, posting, sending)
3. NEVER pass --stdin with body content to any command
4. NEVER call webhook execute, message create, channel post, or any mutation endpoint
5. Timeout: 10 seconds per call, 2 minutes total for all testing
6. Stop immediately on 401/403 (don't burn rate limits on bad auth)
7. Print every command to stderr BEFORE executing it
8. Use --limit 1 on all list calls (minimize API usage)
9. Use --max-pages 5 on sync (tiny scope)

## Test Sequence

1. Set the API key as env var: `export <ENV_VAR_NAME>="<key>"`
2. `<cli> doctor` - validates auth works (expect 200 OK)
3. Pick 3 list endpoints, run each with `--limit 1 --json`
4. From the first list result, extract one ID
5. Run `<cli> <resource> get <id> --json` to validate single-resource fetch
6. If data layer exists: `<cli> sync --max-pages 5` to validate sync with tiny scope
7. If search exists: `<cli> search "a" --limit 1` to validate search
8. Report results:

```
LIVE API TEST RESULTS
=====================
Auth:     PASS/FAIL (doctor response)
List:     N/M passed (resource names)
Get:      PASS/FAIL (resource + ID)
Sync:     PASS/FAIL (pages synced, blocks synced) or SKIPPED
Search:   PASS/FAIL (result count) or SKIPPED
Parsing:  N errors (list any JSON parsing failures)

Verdict:  PASS/WARN/FAIL
```

### Step 5.5g: Data Pipeline Smoke Test

After sync, verify data actually flowed through:

1. Query entity counts:
   ```bash
   <cli> sql "SELECT 'pages' as t, COUNT(*) as n FROM pages
              UNION ALL SELECT 'blocks', COUNT(*) FROM blocks
              UNION ALL SELECT 'users', COUNT(*) FROM users"
   ```
2. **If ANY primary entity (from Phase 0.7) has 0 rows:**
   - Verdict: **WARN** (not PASS)
   - Report: "Sync completed but [entity] has 0 rows. Possible causes: integration permissions, empty workspace, or sync bug."
   - Suggest: "Share at least one resource with the integration and re-run sync."
3. **If primary entities have rows, test the read path:**
   - `<cli> search "a" --limit 1` should return results
   - `<cli> stale --days 9999` should return all pages
   - `<cli> health` should show non-zero totals
4. **If search returns 0 results but rows exist:**
   - FTS5 indexing is broken. This is a FAIL, not a WARN.

**"0 rows synced" is NEVER a PASS.** A pipeline that moves no data is not tested.

If ANY test fails (WARN or FAIL verdict), automatically enter Phase 5.7 Ship Loop:

1. For each failure, classify the bug:
   - **Auth failure (401/403)**: Check client.go auth header format against spec's securitySchemes
   - **Path not found (404)**: Check the URL path in the command file against the spec
   - **Parse error**: Check response struct tags against actual API response shape
   - **Sync failure**: Check pagination params, cursor handling, rate limiting
   - **Timeout**: Check if the endpoint exists and the base URL is correct

2. Write a targeted fix plan listing each bug with file:line and proposed fix
3. Present the plan to the user: "Live testing found N bugs. Here's the fix plan. Proceed?"
4. If yes: fix all bugs (delegate each code-writing patch to Codex if `CODEX_MODE` is true, otherwise fix directly in Claude), then re-run Phase 5.5 live tests to verify
5. If all tests pass after fix: proceed to Final Report with PASS
6. If still failing: report remaining issues, max 2 fix cycles for live test bugs

---

# PHASE 5.7: SHIP LOOP

This phase runs automatically when Phase 5.5 live tests find bugs, OR when the user asks "is this shippable?"

## Auto-trigger from Phase 5.5

When live API testing finds bugs, this phase runs immediately (no user prompt needed).
The fix plan is derived directly from the test failures - concrete bugs with concrete fixes.

## Auto-trigger from "is this shippable?"

When the user asks "is this shippable?", "can we ship this?", "is it ready?", or similar:

1. Run the Quality Scorecard + Proof of Behavior verification
2. If API key is available: also run Phase 5.5 live tests
3. Collect all issues into a single list
4. If PASS (score >= 65, no critical issues, live tests pass): "Yes, ship it. Quality Score: X/100."
5. If WARN (minor issues only): "Shippable with caveats: [list]. Quality Score: X/100."
6. If FAIL (critical issues):
   - Present: "Not yet. Found N issues. Top 3:"
   - List each issue with severity, file, and proposed fix
   - Ask: "Want me to fix these and re-test?"
   - If yes: write fix plan -> apply fixes (delegate code-writing patches to Codex if `CODEX_MODE` is true) -> re-run verification + live tests -> present updated score
   - If no: present issues for manual review

## Fix Loop Rules

- Max 3 fix-loop iterations per session
- Each iteration targets only the top 3 highest-impact issues
- If `CODEX_MODE` is true, each fix is a separate Codex call using the same Codex Delegation Pattern as Phase 4 and Phase 4.9
- Claude still owns the bug triage, fix plan, verification, and go/no-go decision; Codex only writes the scoped patch
- After each fix: `go build ./... && go vet ./...` must pass
- After each fix: re-run Proof of Behavior verification
- After each fix: if API key available, re-run live tests
- After 3 iterations: report remaining issues and stop (avoid infinite loops)
- Each iteration should show: score before -> score after -> delta

---

# PHASE 5.9: OFFER EMBOSS

After presenting the final report (Phase 5), use AskUserQuestion to ask:

"The CLI scored [X]/100 (Grade [Y]). Want me to run an emboss pass to improve it further? This re-researches the landscape, finds the top 5 improvements, builds them, and re-scores."

Options:
- "Yes, run emboss" -> proceed to Emboss Mode (top of this skill)
- "No, I'm done" -> end the run
- "I'll emboss later" -> tell user they can run `/printing-press emboss ./<api>-pp-cli`

**WAIT for the user's answer before proceeding.** Do NOT continue or end the run until answered. Emboss is a FOLLOW-UP, not an automatic step. The user decides.

---

## Writing Specs from Docs

When no OpenAPI spec exists:

1. **WebFetch** the API docs
2. **Read** `~/cli-printing-press/skills/printing-press/references/spec-format.md`
3. Read the docs and identify EVERY endpoint
4. Write YAML spec to `/tmp/<api>-spec.yaml`
5. Generate from it

You ARE the brain. Read the docs yourself and write the spec.

## Submit to Catalog

`/printing-press submit <name>` - gather metadata, write `catalog/<name>.yaml`, create PR.

## Safety Gates

- Preview before generating
- Output directory conflict: check before overwriting
- Untrusted specs: note if not from known-specs registry
- Max 3 retries on quality gate failure

## External Tool Interference

Linters, formatters, and pre-commit hooks may modify files during the session. This is expected — but be aware:

- **After any external modification to a file you depend on, re-read it before writing dependent code.** A linter may change method signatures (e.g., `GetSyncState` returning 2 values → 4 values), add imports, rename variables, or restructure functions. Code you write against the pre-linter version will fail to compile.
- **Check the system-reminder after each tool call.** Claude Code shows diffs from external modifications in system reminders — read them.
- **If a file was modified by a hook, the hook's version wins.** Don't fight it by reverting to your version. Adapt your dependent code to match the new signatures.

## Scorecard Limitations

The scorecard measures file patterns, not behavior. Known blind spots:

- **sync_correctness** and **data_pipeline_integrity** hardcode the filename `sync.go` — sync logic in other files (e.g., `channel_workflow.go`, `sync_cmd.go`) scores 0.
- **workflows** and **insight** use narrow prefix lists biased toward project-management APIs — scheduling, payment, and communication workflow commands may not match.
- **dead_code** produces false positives when flags are passed via struct rather than accessed directly.

**When the scorecard and verify disagree, verify is more authoritative.** A CLI that scores 57/100 on the scorecard but passes 91% of runtime tests is better than one that scores 80/100 but crashes on first use. Report both numbers. Don't chase scorecard points at the expense of actual behavior.

## Anti-Shortcut Rules

These phrases indicate a phase was shortcut. If you catch yourself writing them, STOP and re-do the phase:

- "This is a limitation of the generator" (fix it, don't accept it)
- "Complex types not supported" (add --stdin examples)
- "We'll skip this for now" (no skipping - do it or explain why it's impossible)
- "The quality is good enough" (score it against the quality bar, prove it's good enough with numbers)
- "Let's wrap up" (are all 5 phases complete with artifacts?)
- "This API doesn't need local persistence" (Did you run Phase 0? Check the data profile. If search need is high, it needs persistence.)
- "This is just an API wrapper" (Run Phase 0 again. What would a thoughtful developer build?)
- "The API is GraphQL-only so we can't use printing-press" (Wrong. Skip the REST generator, hand-write commands with a GraphQL client in Phase 4. Every other phase runs normally.)
- "I'll polish the README instead of building workflows" (Phase 4 Priority 1 is workflows. README is Priority 3. Do not skip ahead.)
- "The Phase 0.5 workflows are future work" (They are the product. Build them now or the CLI is just an API wrapper.)
- "316 commands is better than 12" (Depth beats breadth - discrawl proves this. But depth means building the RIGHT 12 commands, not the same 12 commands for every API. Check the competitor feature matrix from Phase 0.6.)
- "The API doesn't need local persistence" (Check data gravity scores from Phase 0.7. If any entity scores >= 8, it needs SQLite with proper columns.)
- "FTS5 is overkill for this API" (If any entity has 2+ text fields AND data gravity >= 8, it needs FTS5. That's how search works.)
- "REST polling is fine for tail" (Check if the API has WebSocket/SSE/Gateway. If yes, use it. REST polling misses events and wastes rate limit budget.)
- "The generic store is good enough" (Domain-native tables ALWAYS beat JSON blob tables. A `messages` table with `channel_id`, `author_id`, `content` columns enables joins and filters that JSON blobs can't. Write the schema.)
- "I'll build the data layer later" (Phase 0.7 runs BEFORE generation. The data layer spec informs Phase 4 Priority 0. Build it first, then workflows use it.)
- "The artifact is just documentation" (The 7 plan artifacts ARE the product. They capture reasoning, evidence, and decisions. The generated CLI is a side effect.)
- "The CLI compiles so it works" (Compilation proves syntax, not semantics. A command that builds can still 400 on every real call. Run the dogfood.)
- "We can't test without API keys" (The OpenAPI spec defines response schemas. Generate mocks from the spec. Test against them. Zero keys needed.)
- "The dry-run looks right" (Dry-run validates request construction. You also need to feed synthetic responses to validate output parsing, --select, and table rendering.)
- "I'll add a helpers.go with the patterns the scorecard checks for" (STOP. Every function in helpers.go MUST be called by at least one command. Dead code is worse than missing code. Phase 4.6 will catch this.)
- "The error handling score is low, let me add error types" (STOP. Error types must be used in actual error paths. Adding newAuthError() that nobody calls is gaming, not engineering.)
- "I'll add --csv and --quiet flags to root.go" (STOP. Every registered flag must be checked in at least one RunE function. Flags nobody reads are dead flags. Phase 4.6 catches this.)
- "I'll add insight command files to match the scorecard prefixes" (STOP. Insight commands must query tables that are actually populated. A health command querying an empty database is theater.)
- "I'll skip the dogfood/verify to save time" (Skipping testing is how you produce a CLI that scores 73/100 with a broken core feature. The GitHub run proved this. Run `printing-press verify`.)
- "The scorecard is 73 so it's good enough" (The scorecard measures files, not behavior. A 73 scorecard with 0% verify pass rate is a CLI that looks good on paper and crashes on first use. Run verify.)
- "I tested 5 commands manually, that's enough" (5/127 is 3.9%. That's not testing. Run `printing-press verify` which tests every command automatically in under 60 seconds.)
- "The CLI compiles so it's ready to ship" (Compilation proves syntax. `printing-press verify` proves behavior. A CLI that compiles but 404s on sync is not shippable.)
- "The generated command names are fine" (They're machine names from operationIds. Normalize them: retrieve-a -> get, post -> create. Phase 4 Priority 3.)
- "The module path is close enough" (It's a Go import path. It must be exact or `go install` fails for everyone.)
- "0 rows synced is still a PASS" (A pipeline that moves no data is not tested. It's WARN at best.)
- "Users can go install it" (Most users don't have the Go toolchain. Add goreleaser. Phase 4 Priority 6.)
- "I chose the name in Phase 0.8" (Choosing isn't applying. Grep for the old name. If it appears, the rename is incomplete.)
- "The scorecard is the objective" (The scorecard measures proxies. The objective is: would a user of the top competitor switch? Check the three-benchmark gate.)
- "We complement the incumbent, we don't compete" (Users don't want two CLIs. If the incumbent has a feature, you need it too - Phase 0.6 table stakes.)
- "That feature is anti-scope" (If a competitor with >100 stars has it, it's not anti-scope. It's a backlog item. Phase 0.6 classification rules.)
- "I'll skip the agent readiness review" (You MUST attempt the dispatch. If the agent isn't installed, the dispatch will fail and you warn the user. If it IS installed, it runs. You don't get to decide it's unnecessary.)
- "The CLI is already agent-native enough" (The scorecard checks for flags. The agent readiness reviewer checks 7 deeper principles — alias correctness, output routing, delete safety gates, exit code semantics. These are different things. Dispatch it.)

**Module path rule:**
- The go.mod module path MUST be a valid Go import path with a real org name (e.g., `github.com/mvanhorn/discord-cli`). The literal string `USER` is never acceptable. The generator auto-derives from git config.

**Time Budget Guidance:**
- Phase 0-1 (Research + Parity Audit): 20% of total time
- Phase 2 (Generate + Normalize Names): 5%
- Phase 3 (Audit + Competitor Comparison): 5%
- **Phase 4 Priority 0 (Data Layer): 15%** - The architecture foundation.
- **Phase 4 Priority 1 (Table Stakes): 15%** - Match the competition. THIS IS THE PRODUCT.
- Phase 4 Priority 2 (Workflows): 10%
- Phase 4 Priority 3-7 (Names, Score, Tests, Distribution): 15%
- Phase 4.5-4.8 (Dogfood + Verify): 10%
- Phase 5 (Final Report + Emboss Offer): 5%

Table stakes features get 15% - the same as the data layer. Because matching the competition IS the product.

**Scorecard uses two tiers (100-point scale):**
- Tier 1: Infrastructure (string-matching, 50 max) - does the skeleton have the right patterns?
- Tier 2: Domain Correctness (semantic, 50 max) - does the code actually work?
- Use `--spec` flag on scorecard command to enable Tier 2 validation against the OpenAPI spec.

**Discrawl benchmark for communication APIs:**
- After Phase 4, ask: "Would a discrawl user switch to this CLI?" If the answer is "no because [feature X]", that's your remaining Phase 4 work.
