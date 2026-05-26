---
decision: 2026-05-25-008-espn-goat-findings
status: structural-validation-complete-agent-dogfood-pending
created: 2026-05-25
plan: docs/plans/2026-05-25-008-feat-espn-goat-self-learning-plan.md
predecessor:
  - docs/decisions/2026-05-25-007-espn-content-cleanup-findings.md
related_plans:
  - docs/plans/2026-05-25-006-feat-machine-self-correction-plan.md (queued; gate is the agent dogfood under this plan)
---

# ESPN GOAT + self-learning — content fixes shipped, agent dogfood pending

Plan 008 distilled the 2026-05-25 seven-session learnings into ESPN-side fixes plus the minimal self-correction surface (`playbook amend --add-note`). All structural validation complete; agent-driven 7-shape rerun is the user's final test before plan 006 (cross-CLI machine self-correction) unblocks.

## What shipped

### U1 — League stopwords + merged playbook

Five league tokens (`mlb`, `nba`, `nfl`, `nhl`, `mls`) added to ESPN's stopword list. Caps and lowercase variants now strip identically, fixing the family-key bug where the user's "NBA" got auto-promoted to entity while seeded examples used lowercase "nba", producing different families. With league names as stopwords, both variants collapse to the same family.

Merged `league_top_bottom_mlb` and `nba_league_top_bottom` into a single `league_top_bottom` playbook. Notes carry per-league division maps for MLB, NBA, NFL, NHL, MLS. One playbook covers all five sports' top/bottom-per-division query family.

### U2 — Notes corrections from the 7-session debug responses

- **season_recap**: dropped the explicit per-category stat-index map. It drifted (`avgRebounds` at general[9] in one session vs general[11] in another). Notes now emphasize the runtime `names.indexOf(<key>)` pattern only, with a clear warning that hardcoded indices WILL go stale across seasons.
- **last_game_stats**: added the summary-endpoint envelope gotcha. ESPN's `summary --event <id>` command wraps its payload in `{meta, results}`; the boxscore-shaped data is at `.results.header`, NOT at `.header`. Carter Bryant and Stephen Curry sessions both burned 2 calls each rediscovering this. Notes now spell it out explicitly.
- **draft_pick**: strengthened the "don't teach the year-specific answer" guard. The 2026-05-25 draft-pick agent fired `teach-playbook` with notes saying "Wizards/2026 #1 pick" despite the existing rule. Notes now carry explicit rationale: a year-stamped answer in shared playbook notes will mislead next year's queries because recall has no answer-decay TTL.

### U3 — SeedVersion bump to `2026-05-25-espn-004`

The sentinel re-seed mechanism (plan 007 U5/U6) now propagates the corrections automatically. Existing user DBs pick up the new content on the next CLI invocation. User-authored playbooks at non-embedded family keys are untouched.

### U4 — New `playbook amend --add-note` command

One-line UPSERT command. Computes the query family via the same Normalize+PromoteEntities path recall uses, looks up the existing `learning_playbooks` row, appends `\n\n[amend YYYY-MM-DDTHH:MMZ]: <text>` to `notes_text`. UPSERT: if no row exists for the family, creates a notes-only one (so cold-start corrections still land).

Silent on success. Errors to teach.log. Same fire-and-forget posture as `teach` and `teach-playbook`. Designed to be the agent's lowest-friction path from "I identified a correction in my debug response" to "the correction is recorded for next time."

Six tests cover: happy path with existing playbook, empty family creates notes-only, required-flag validation, multiple amends accumulate with distinct timestamp markers, NO_LEARN env respected.

### U5 — SKILL.md Step 6

New step in the existing decision tree: "After answering the user, if your debug-protocol response identified a concrete correction the notes should know — workaround, undocumented endpoint shape, stale field name, observed schema drift — fire `playbook amend --query "..." --add-note "..." &` BEFORE emitting your user-facing response."

Worked examples explicitly distinguish "worth amending" (summary endpoint envelope, postseason mode fallback) from "NOT worth amending" (year-specific answers, per-team data the playbook already retrieves).

### Discovered during execution — multi-family install

When U6 dogfood-verified the merged playbook, the NBA query "best 3 NBA teams in each division" still missed because the install path only derived a SINGLE family from the first `query_family_examples` entry. With the merged playbook, "top" / "best" / "bottom" / "worst" produce different families. Fixed: `installPlaybooksFromEmbed` now iterates ALL example queries and seeds the playbook content under each distinct family. SeedVersion bumped to 004. End-to-end verified: NBA / MLB / bottom-variant all hit their own families surfacing the same playbook content + notes.

## Structural validation (fresh $HOME, freshly installed binary)

15 rows installed (1 sentinel + 14 family entries across 5 playbooks):

| Query family | Source playbook | Notes |
|---|---|---|
| `__seed_meta__` | (sentinel) | tracks SeedVersion |
| `end season` | season_recap | "how did X end the season" |
| `end led season` | season_recap | variant |
| `end led ppg rpg season spg` | season_recap | full phrasing |
| `next play` | team_today | next-game family |
| `3 division each teams top` | league_top_bottom | MLB "top 3" |
| `3 best division each teams` | league_top_bottom | NBA "best 3" |
| `3 division each teams worst` | league_top_bottom | "worst 3" |
| `3 bottom division each teams` | league_top_bottom | "bottom 3" |
| `5 division each teams top` | league_top_bottom | NFL "top 5" |
| `last` | last_game_stats | single-token degenerate |
| `draft pick upcoming` | draft_pick | variant |
| `draft lottery next` | draft_pick | variant |
| `1 draft pick` | draft_pick | variant |
| `#1 draft draft pick upcoming` | draft_pick | full phrasing |

Cross-checks (all green):
- NBA query "who are the best 3 NBA teams in each division" → hits family `3 best division each teams`, playbook + notes attached, notes carry BOTH MLB and NBA division maps.
- MLB query "top 3 mlb teams in each division" → hits family `3 division each teams top`, same playbook + notes.
- Bottom variant "worst 3 nba teams in each division" → hits family `3 division each teams worst`, same playbook + notes.
- `playbook amend --query "..." --add-note "..."` → appends `[amend YYYY-MM-DDTHH:MMZ]: <text>` to the matching family's notes verifiably.

Tests: 263 pass across 14 packages.

## What this doc cannot prove (the user's final dogfood)

The remaining test is whether agents in actual Claude Code sessions:

1. Fire `playbook amend` when their debug response identifies a correction (R9: ≥1 of 7 sessions fires amend).
2. Don't burn calls rediscovering the summary envelope (`.results.header`) thanks to the new notes.
3. Don't burn calls rediscovering the byathlete schema thanks to the runtime-indexOf-only pattern.
4. Hit the NBA playbook directly (R9 critical case — was the broken one).

The 7 transcript shapes:
1. Who has the #1 draft pick in the upcoming draft
2. who are the best 3 NBA teams in each division ← critical, was broken
3. top 3 mlb teams in each division
4. Stephen Curry last game
5. Carter Bryant last game
6. how did Lakers end the season who led in ppg rpg spg
7. how did Pistons end the season who led in ppg rpg spg

Plus a debug protocol prompt after each. The user runs these in fresh Claude Code windows; pastes the scorecards back.

## Plan 006 go/no-go (post-this-dogfood)

If the user's 7-shape rerun shows:
- ≥5 of 7 measurably improved over the prior 2026-05-25 evening baseline
- NBA query hits the playbook (no cold-start)
- ≥1 of 7 sessions fires `playbook amend` with a real correction during the dogfood

→ Plan 006 (machine self-correction with auto-capture trace + amend-suggestion surface + cross-shape `related_playbook`) unblocks. Cross-CLI port for prediction-goat / kalshi / polymarket etc. queues as follow-up.

If amend works but no agent fires it during dogfood despite identifying corrections in debug responses → plan 006's auto-suggestion surface ("previous session ran 9 calls vs expected 4, consider amending") becomes the load-bearing piece. Manual amend isn't enough discipline pressure on its own.

If wall-time wins don't hold OR NBA still misses → there's a deeper issue with how the loop integrates with how agents actually work, and we route to /ce-brainstorm before more dogfood.

## Files changed in this plan

ESPN library (`mvanhorn/printing-press-library`):
- Modified: `library/media-and-entertainment/espn/internal/cli/learn_init.go` (stopwords)
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/season_recap_notes.md` (drop index map, emphasize indexOf)
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/last_game_stats_notes.md` (summary envelope gotcha)
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/draft_pick_notes.md` (don't-teach-answer guard)
- Deleted: `library/media-and-entertainment/espn/internal/cli/playbooks/league_top_bottom_mlb.json` + `_notes.md`
- Deleted: `library/media-and-entertainment/espn/internal/cli/playbooks/nba_league_top_bottom.json` + `_notes.md`
- New: `library/media-and-entertainment/espn/internal/cli/playbooks/league_top_bottom.json` + `_notes.md`
- Modified: `library/media-and-entertainment/espn/internal/cli/playbooks/embed.go` (SeedVersion 003 → 004)
- Modified: `library/media-and-entertainment/espn/internal/cli/playbook_init.go` (multi-family seed)
- Modified: `library/media-and-entertainment/espn/internal/cli/teach_playbook.go` (newPlaybookAmendCmd)
- Modified: `library/media-and-entertainment/espn/internal/cli/teach_playbook_test.go` (6 amend tests)
- Modified: `library/media-and-entertainment/espn/internal/cli/playbook_init_test.go` (merged-playbook expectation)
- Modified: `library/media-and-entertainment/espn/internal/cli/root.go` (pass learnCfg to playbook)
- Modified: `library/media-and-entertainment/espn/SKILL.md` (Step 6 amend instruction)

Generator (`mvanhorn/cli-printing-press`):
- New: this retrospective doc.
- No template changes (deferred to plan 006 cross-CLI port when next CLI needs it).
