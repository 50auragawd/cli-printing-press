---
decision: 2026-05-25-007-espn-content-cleanup-findings
status: structural-validation-complete-agent-dogfood-pending
created: 2026-05-25
plan: docs/plans/2026-05-25-007-fix-espn-playbook-content-and-auto-install-plan.md
predecessor:
  - docs/decisions/2026-05-25-005-espn-playbooks-findings.md
related_plans:
  - docs/plans/2026-05-25-006-feat-machine-self-correction-plan.md (queued; gate is the next agent dogfood)
---

# ESPN content cleanup + auto-install — structural validation, agent dogfood pending

Plan 007 corrected the ESPN playbook content the 2026-05-25-evening dogfood transcripts surfaced, plus shipped a CLI-agnostic auto-install path so playbooks land in user DBs on first invocation. This doc captures what landed, the structural-validation evidence, and what the user's next agent-driven dogfood needs to confirm before plan 006 (machine self-correction) unblocks.

## What shipped

### U1. season_recap content correction

Notes rewrote the byathlete payload schema explanation. Real shape:

- Per-athlete `categories[i]` carries only `name`, `displayName`, `totals[]`, `values[]`, `ranks`. NO `names[]` or `labels[]` per athlete.
- Schema lives at TOP-LEVEL `categories[]` block: `categories[i].names[j]` (machine key like `"avgPoints"`) and `categories[i].labels[j]` (display label). Index `j` aligns positionally with per-athlete `totals[]`.
- Team filter field is `teamShortName` (e.g., `"GSW"`, `"LAL"`) or numeric `teamId`. NOT `team.abbreviation` (no nested object).
- Added explicit per-category index map for NBA general/offensive/defensive blocks plus the runtime safety pattern (`names.indexOf(<wanted>)`).
- expected_tool_calls reduced from 4 to 2 (skip the teams-object call; standings + byathlete is the minimal path).

### U2. last_game_stats postseason fallback

The Stephen Curry case proved compare/teams choreography dead-ends when the league is in postseason mode and the athlete's team isn't in playoffs (compare returns empty team; teams<id> returns events:[]). Notes now carry the fallback explicitly: scoreboard --dates <regseason-end-window> filtered by team abbreviation. Optional JSON step with `fallback_for` field documents the trigger condition.

### U3. NBA league_top_bottom playbook

New file paired with the renamed league_top_bottom_mlb. NBA division map (Atlantic/Central/Southeast/Northwest/Pacific/Southwest with team abbreviations). Same shape as MLB version — one standings call + client-side group + rank. Family-key now contains the league token so MLB and NBA queries don't collide.

### U4. draft_pick playbook

Notes-heavy playbook documenting the WebFetch route (`espn.com/<sport>/draft/rounds` or `/draft/order`). ESPN's news/search surfaces don't carry draft order. Steps are a single advisory client_side: web_fetch. Notes also document answer decay (lottery results change yearly) and cross-sport applicability.

### U5/U6. Hand-authored auto-install (ESPN only)

Three new files in `library/media-and-entertainment/espn/`:

- `internal/cli/playbooks/embed.go` — `//go:embed *.json *.md` + `SeedVersion` constant
- `internal/cli/playbooks/MANIFEST.md` — stub keeping embed pattern matching even when no playbook content exists; documents the cross-CLI port pattern
- `internal/cli/playbook_init.go` — `runPlaybookInitOnce` reads embed.FS, pairs JSON+notes, calls `UpsertPlaybook` per file. Sentinel row tracks seed version; mismatch triggers re-seed on binary upgrade. Concurrent-safe (sync.Once + DB upsert).
- `internal/cli/root.go` — `runPlaybookInitOnce(cmd.Context())` wired into `PersistentPreRunE` alongside `runLearnInitOnce`, gated by `!flags.noLearn && !shouldSkipLearnHook(...)`.

Plus the test file with 3 tests: seeds all shipped playbooks (with sentinel + content-correctness spot-checks), idempotent re-install, concurrent-safe under 5 goroutines.

Generator-template work was deferred. The user explicitly asked to avoid hardcoding while keeping this ESPN-scoped. The hand-authored shape is mechanically copy-able to other CLIs (rename the SeedVersion CLI suffix, swap the embed package path, swap defaultDBPath argument). Cross-CLI templating waits for the next-CLI-actually-needs-it moment.

## Structural validation (what I verified end-to-end before signing off)

Fresh `$HOME` + `go install`-ed binary:

```
$ espn-pp-cli --version
espn-pp-cli 1.0.0

$ espn-pp-cli playbook list --json
7 rows in fresh DB:
  family=__seed_meta__                                       playbook=no   notes=yes
  family=team_today                                          playbook=yes  notes=yes
  family=end led ppg rpg season spg                          playbook=yes  notes=yes
  family=3 best division each nba teams                      playbook=yes  notes=yes
  family=3 division each mlb teams top                       playbook=yes  notes=yes
  family=last                                                playbook=yes  notes=yes
  family=#1 draft draft pick upcoming                        playbook=yes  notes=yes
```

Recall surfaces playbook + slot resolution for the 5 main families:

| Query | Family hit | Slot resolved | Steps | Expected calls |
|---|---|---|---|---|
| how did Lakers end the season who led in ppg rpg spg | end led ppg rpg season spg | $TEAM -> Los Angeles Lakers | 4 | 2 |
| top 3 mlb teams in each division | 3 division each mlb teams top | - | 3 | 1 |
| best 3 nba teams in each division | 3 best division each nba teams | - | 3 | 1 |
| Stephen Curry last game | last | - | 5 | 3 |
| who has the #1 draft pick in the upcoming draft | #1 draft draft pick upcoming | - | 1 | 1 |

NBA family now hits its OWN playbook (not the MLB one). Cross-entity replay still works (Lakers binds to LA Lakers from the Warriors-seeded family).

## What this doc cannot prove (the user's next dogfood)

Structural validation says the surface FIRES. It does NOT yet prove agents do measurably better work with the corrected content because that requires real Claude sessions running through the seven 2026-05-25-evening shapes. The user's next-step prompt: rerun the same seven queries and capture tool-call count + wall-time per family vs the prior baseline.

The specific hypotheses being tested:
- Warriors / Pistons / Lakers: with the corrected schema notes, does the Warriors session no longer burn 4 calls rediscovering the categories schema?
- Stephen Curry: with the postseason fallback in notes, does the agent route to scoreboard immediately instead of dead-ending in compare+teams?
- NBA best 3: with the new NBA-specific playbook, does the agent skip cold-discovery and apply the division-map directly?
- Draft pick: with the new playbook, does the agent skip the ESPN news/search dead-ends and go straight to WebFetch?

These are the four cases the 7-shape baseline scored COST or BREAK-EVEN on. If U7's agent dogfood shows them flipping to WIN, plan 006 (machine self-correction) becomes the next force-multiplier. If they're still flat, plan 006's amend-suggestion surface is the right next step (the notes still need correcting in real sessions; tooling has to make that nearly free).

## Cross-CLI port template (for the moment another CLI needs this)

For any future PP CLI to inherit auto-install playbooks:

1. Create `internal/cli/playbooks/` directory in the new CLI's library tree.
2. Copy `embed.go` from ESPN; rename `SeedVersion` constant suffix to the new CLI name.
3. Copy `MANIFEST.md` from ESPN (the stub + convention doc).
4. Copy `internal/cli/playbook_init.go` from ESPN; replace the four import paths and the `defaultDBPath` argument with the new CLI name; replace stderr-warning prefix.
5. Add `runPlaybookInitOnce(cmd.Context())` to the new CLI's root.go PersistentPreRunE alongside `runLearnInitOnce`.
6. Author the per-CLI JSON+MD pairs in the new CLI's playbooks directory. Use lowercase tokens in `query_family_examples` so the entity extractor's ALL-CAPS rule doesn't auto-promote league/sport tokens and strip them from the derived family key (the lesson U5 surfaced when seeding NBA queries).

Generator-template integration: queue as a follow-up when 2+ CLIs need the same install logic. Until then, mechanical copy is fine.

## Plan 006 go/no-go

Decision deferred to the next agent dogfood. If the user's seven-shape rerun shows ≥4 of 7 families measurably improve in either tool-call count or wall-time:

- ≥4 improve in both metrics: ship plan 006 (machine self-correction). The corrected content + auto-install proves the loop delivers; 006 multiplies value across other CLIs by making self-correction nearly free.
- ≥4 improve in tool-call count but wall-time stays flat: ship plan 006 with extra emphasis on agent-verification-reduction (a SKILL.md section telling agents to trust warm hits more aggressively).
- <4 improve: pause. The content fixes + auto-install were necessary but not sufficient. Route to /ce-brainstorm to evaluate whether the playbook concept needs deeper rework before more dogfood.

## Files added/changed in PR #851

Library (`mvanhorn/printing-press-library`):
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/season_recap.json` + `_notes.md`
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/last_game_stats.json` + `_notes.md`
- Renamed: `league_top_bottom.json`+ `_notes.md` -> `league_top_bottom_mlb.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/nba_league_top_bottom.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/draft_pick.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/embed.go`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/MANIFEST.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbook_init.go` + `_test.go`
- Modified: `library/media-and-entertainment/espn/internal/cli/root.go` (call runPlaybookInitOnce)

Generator (`mvanhorn/cli-printing-press`): no changes this plan (deferred to next-CLI port moment).
