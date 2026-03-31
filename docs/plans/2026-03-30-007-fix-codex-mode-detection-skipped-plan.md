---
title: "fix: Codex mode detection skipped during skill execution"
type: fix
status: active
date: 2026-03-30
---

# fix: Codex mode detection skipped during skill execution

## Overview

Merge the Codex Mode Detection bash block into the Setup Contract so both run as a single mandatory block. Currently they're separate sections and the LLM skips the second one.

## Problem Frame

When running `/printing-press Redfin Codex`, the skill executor ran the setup contract (binary check, version, directories) but skipped the Codex Mode Detection block entirely. It jumped straight to Phase 0 without checking for the `codex` keyword. The user noticed mid-run that codex delegation never activated.

Root cause: the setup contract ends at line 151 (`PRESS_SETUP_CONTRACT_END`), then a new `### Codex Mode Detection` section starts at line 153 with its own bash block. The LLM treats these as two separate steps and skips the second one when eager to start Phase 0. There's no enforcement that forces both blocks to execute.

## Requirements Trace

- R1. Codex detection must run every time the setup contract runs - not as a separate optional step
- R2. The `CODEX_MODE` variable and circuit breaker state must be initialized before Phase 0 begins

## Scope Boundaries

- Only modifying `skills/printing-press/SKILL.md`
- Not changing codex delegation behavior, just ensuring detection actually runs
- Not changing the setup contract's other functionality

## Key Technical Decisions

- **Merge into one bash block**: Move the codex detection code inside the setup contract's `PRESS_SETUP_CONTRACT_START`/`END` markers. One block = one execution. Rationale: LLMs reliably execute single code blocks marked "before doing anything else." They skip secondary blocks that look optional.
- **Keep the `### Codex Mode Detection` heading as prose**: Move it above the merged block as explanatory text, not a separate section with its own code block.

## Implementation Units

- [ ] **Unit 1: Merge codex detection into setup contract**

  **Goal:** Make codex detection impossible to skip by merging it into the single mandatory setup block.

  **Requirements:** R1, R2

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - Move the codex detection bash code (lines 157-184) into the setup contract bash block (before line 150, after `mkdir -p`)
  - Remove the separate `### Codex Mode Detection` section header
  - Remove the `CODEX_DETECTION_START`/`END` HTML markers (no longer needed as a separate block)
  - Keep the `PRESS_SETUP_CONTRACT_START`/`END` markers wrapping everything
  - Add a comment in the merged block like `# --- Codex mode detection ---` for readability

  **Patterns to follow:**
  - The setup contract already has inline comments for each section (binary check, scope derivation, directory creation)

  **Verification:**
  - Single bash block between `PRESS_SETUP_CONTRACT_START` and `PRESS_SETUP_CONTRACT_END`
  - No separate codex detection bash block exists
  - `CODEX_MODE` and `CODEX_CONSECUTIVE_FAILURES` are initialized in the merged block
  - The text after the block still references version checking and run-scoped paths as before
