---
status: active
created: 2026-05-24
revised: 2026-05-25
type: feat
plan_depth: lightweight
primary_repo: mvanhorn/printing-press-library
predecessor_plan: docs/plans/2026-05-23-002-feat-generator-wide-self-learning-cli-plan.md
gates_phase: 2.5 (between Phase 2 ship and Phase 3 full sweep of predecessor plan)
---

# feat: ESPN learn config — first real-seed pilot (patch path)

**Target repo:** `mvanhorn/printing-press-library` (the published library — patch path, not regenerate path).

## Why this plan exists

The predecessor plan shipped the self-learning loop into 5 pilot CLIs with empty `spec.Learn` defaults — recall returns `{found: false}` because there are no entity-lookup seeds. The dogfood window the predecessor plan called for (1-2 weeks of recall/teach traffic) measures nothing in that state.

This plan populates ESPN's `internal/cli/learn_init.go` directly in the library with real seed data (NFL/NBA/MLB/MLS team rosters), so the pilot actually exercises the entity-substitution + pattern-extraction layer the predecessor plan built.

## Path decision: patch, not regenerate

**Patch the published artifact directly.** ESPN was originally generated from `/tmp/espn-spec.yaml` (transient, gone). There's no `catalog/espn.yaml`, no canonical source spec anywhere in cli-printing-press. Regenerating means reconstructing the source spec from the published artifact first — hours of work and divergence risk from the existing CLI's hand-patches.

Patch path:
1. Hand-edit `library/media-and-entertainment/espn/internal/cli/learn_init.go` with real seeds
2. Record the patch in `library/media-and-entertainment/espn/.printing-press-patches.json` per AGENTS.md convention
3. Ship as a library-side PR

**Known risk:** the sweep tool doesn't currently read patches.json to preserve hand-edits. If anyone re-runs `tools/sweep-learn-install/` against espn later (e.g., during U15 full sweep), the hand-edited `learn_init.go` gets overwritten back to stub-defaults. Worst case: seeds get lost, recall falls back to benign no-op, no breakage but value loss. Acceptable for the pilot; side-car infrastructure can be designed later if mass re-sweeps become a real recurring pain.

## Scope

### In scope

- Hand-edit `library/media-and-entertainment/espn/internal/cli/learn_init.go` to populate:
  - `newLearnConfig()` with the ESPN ticker patterns (regexes for ESPN event IDs, athlete IDs, team IDs) and sports stopwords
  - `initLearn()` with the entity-lookup seed map: ~120 team entries across NFL (32), NBA (30), MLB (30), MLS (30) with 2-4 aliases each
- The seed data is already authored in the worked example at `cli-printing-press:docs/SPEC-LEARN-AUTHORING.md` — copy from there into Go code form
- Add a `.printing-press-patches.json` entry for this customization
- Validate locally: `go build ./...`, `go test ./...`, smoke recall with fresh HOME
- Open a per-CLI PR against `printing-press-library` main

### Out of scope

- Reconstructing espn's source spec in cli-printing-press (catalog/espn.yaml authoring)
- Regenerating espn via `cli-printing-press generate espn`
- Side-car file infrastructure (sweep tool reading `.learn-seeds.yaml` per-CLI)
- The other 4 pilots (contact-goat, company-goat, podcast-goat, instacart) — same exercise, separate plans
- Auto-populating seeds from a CLI's `sync` data

## Implementation Units

### U1. Author ESPN's spec.Learn data as Go code

**Goal:** Convert the YAML-shaped seeds in the authoring guide into the Go literal map shape that `initLearn()` and `newLearnConfig()` expect.

**Files (in `mvanhorn/printing-press-library`):**
- Modify: `library/media-and-entertainment/espn/internal/cli/learn_init.go`

**Source data (in `mvanhorn/cli-printing-press`):**
- Read: `docs/SPEC-LEARN-AUTHORING.md` (contains all 122 team entries with aliases as YAML)

**Approach:**

The current `learn_init.go` is the stub-emit version from the sweep tool — `newLearnConfig()` returns an unconfigured `entities.NewConfig()`, `initLearn()` is a no-op. Replace with:

```go
// Illustrative — actual function signatures come from the generator's
// learn_init.go.tmpl. Mirror existing struct shape.

func newLearnConfig() *entities.Config {
    cfg := entities.NewConfig()
    cfg.RegisterTickerPattern(regexp.MustCompile(`^[0-9]{9}$`))
    cfg.RegisterTickerPattern(regexp.MustCompile(`^a-[0-9]+$`))
    cfg.RegisterTickerPattern(regexp.MustCompile(`^[a-z]{2,4}-[a-z]+$`))
    cfg.RegisterStopwords("vs", "v", "versus", "game", "games", "match",
        "matches", "tonight", "today", "yesterday", "tomorrow", "weekend",
        "schedule", "scoreboard", "score", "scores", "result", "results",
        "winner", "stats", "standings", "lineup", "roster")
    return cfg
}

func initLearn(ctx context.Context, db *sql.DB) error {
    seeds := map[string][]lookups.ConfigSeed{
        "nfl_team": {
            {Canonical: "Arizona Cardinals", Aliases: []string{"Cardinals", "Cards", "ARI", "AZ"}},
            {Canonical: "Atlanta Falcons",   Aliases: []string{"Falcons", "ATL"}},
            // ... 30 more NFL teams ...
        },
        "nba_team": { /* 30 teams */ },
        "mlb_team": { /* 30 teams */ },
        "mls_team": { /* 30 teams */ },
    }
    return lookups.SeedFromConfig(ctx, db, seeds)
}
```

The exact identifier types (`lookups.ConfigSeed`, `entities.Config`, etc.) and function signatures live in the sweep-emitted file — read it before editing to match the shape.

Pull the full team list from `cli-printing-press:docs/SPEC-LEARN-AUTHORING.md`. The YAML format `{canonical: "...", aliases: ["...", "..."]}` maps directly to `{Canonical: "...", Aliases: []string{"...", "..."}}`.

### U2. Record the patch in `.printing-press-patches.json`

**Goal:** Catalog the customization per AGENTS.md convention.

**Files:**
- Create or modify: `library/media-and-entertainment/espn/.printing-press-patches.json`

**Approach:**

Per AGENTS.md, add an entry:

```json
{
  "id": "espn-learn-seeds",
  "summary": "Populate learn_init.go with NFL/NBA/MLB/MLS team rosters and ESPN ticker patterns; sweep tool stub-emits empty defaults.",
  "reason": "Pilot of plan 2026-05-23-002 ships the self-learning loop with empty defaults. Adding real seeds unlocks entity-aware recall + pattern extraction for ESPN's killer flow ('Niners game tonight' → resolves to right event via alias). Future re-sweeps will overwrite this until side-car infrastructure lands.",
  "files": ["internal/cli/learn_init.go"],
  "validated_outcome": "recall 'Cowboys game tonight' (never directly taught) returns the right event via nfl_team alias resolution after teaching 'Niners game tonight'."
}
```

Use the existing patches.json structure if one is already there (it might be after the previous sweep). Don't overwrite — append to `patches[]`.

### U3. Validate locally + open library PR

**Goal:** Confirm the change works end-to-end + ship.

**Approach:**

```bash
cd ~/printing-press-library
git checkout -b feat/espn-learn-seeds main  # or stack on feat/learn-pilot-sweep if #826/#827 not yet merged
# After editing learn_init.go and patches.json:
cd library/media-and-entertainment/espn
go build ./...
go test ./...
go build -o /tmp/espn-pp-cli ./cmd/espn-pp-cli

# Smoke with fresh HOME (critical — per predecessor plan's retrospective)
HOME=/tmp/espn-test-$$ /tmp/espn-pp-cli teach \
  --query "Niners game tonight" \
  --resource <some-real-espn-event-id> \
  --resource-type events

HOME=/tmp/espn-test-$$ /tmp/espn-pp-cli recall "49ers game tonight" --json
# Expected: hits the same resource via "49ers" alias → "San Francisco 49ers" canonical

HOME=/tmp/espn-test-$$ /tmp/espn-pp-cli recall "Cowboys game tonight" --json
# Expected: after pattern extraction fires (3+ similar teaches), substitutes via nfl_team
# OR: returns no-hit if pattern threshold not reached yet — both acceptable

cd ~/printing-press-library
git add library/media-and-entertainment/espn/
git commit -m "feat(cli): populate espn learn_init with NFL/NBA/MLB/MLS rosters"
git push -u origin feat/espn-learn-seeds
gh pr create --title "feat(cli): populate espn learn_init.go with real team rosters" \
  --body "Patch path per docs/plans/2026-05-24-001..."
```

If `#826` and `#827` haven't merged yet, stack this PR on `feat/learn-pilot-sweep` (PR #827's branch) so the espn changes flow through naturally with the rest of the pilot. Otherwise base on main.

## Verification

Done when:
- ESPN's `learn_init.go` carries the 122-team roster + ESPN ticker patterns + sports stopwords
- `.printing-press-patches.json` has the new patch entry
- Local validation passes (build, test, smoke teach + recall with fresh HOME)
- PR is open against `mvanhorn/printing-press-library`

## Pickup prompt

```
/ce-work /Users/mvanhorn/cli-printing-press/docs/plans/2026-05-24-001-feat-espn-learn-config-pilot-plan.md
```

The seed data lives at `~/cli-printing-press/docs/SPEC-LEARN-AUTHORING.md` (NFL/NBA/MLB/MLS rosters with aliases). The current stub-emit `learn_init.go` lives at `~/printing-press-library/library/media-and-entertainment/espn/internal/cli/learn_init.go`. AGENTS.md at `~/printing-press-library/AGENTS.md` covers the patches.json convention.

Note: this is a patch path, not a regenerate path. Don't reconstruct espn's source spec; just edit the published artifact directly and record the customization in patches.json.
