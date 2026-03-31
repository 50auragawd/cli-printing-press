---
title: "fix: Phase 6 (Publish) skipped after shipcheck"
type: fix
status: active
date: 2026-03-30
---

# fix: Phase 6 (Publish) skipped after shipcheck

## Overview

Add a mandatory checkpoint after Phase 5.5 (Archive) that forces the LLM to evaluate Phase 6 (Publish). The LLM currently prints a summary report after archiving and treats that as the end of the run, never offering to publish.

## Problem Frame

In the Redfin run, the skill executor completed Phase 4 (Shipcheck: 81/100, ship-with-gaps), Phase 5 (skipped live smoke), Phase 5.5 (archived manuscripts), then printed a summary and stopped. Phase 6 (Publish) exists in the skill with clear instructions to offer publish via AskUserQuestion, but the LLM never reached it.

Same pattern as codex detection and sniff gate: after producing a large output (the summary report), the LLM treats the conversation as finished. The publish phase exists in the file but gets silently skipped.

## Requirements Trace

- R1. After archiving, the LLM must evaluate Phase 6 (Publish) for every run with a ship or ship-with-gaps verdict
- R2. The user must be asked whether to publish via AskUserQuestion - the run should not end silently

## Scope Boundaries

- Only modifying `skills/printing-press/SKILL.md`
- Not changing publish behavior, just ensuring the offer happens

## Implementation Units

- [ ] **Unit 1: Add mandatory publish checkpoint after archive**

  **Goal:** Force the LLM to evaluate Phase 6 after archiving.

  **Requirements:** R1, R2

  **Files:**
  - Modify: `skills/printing-press/SKILL.md`

  **Approach:**
  - After the Phase 5.5 archive bash block (around line 1516), add a bolded mandatory checkpoint:
    "MANDATORY: After archiving, you MUST proceed to Phase 6 (Publish). Do not print a summary and stop. Do not treat archiving as the end of the run. The run ends when the user has been asked about publishing."
  - At the top of Phase 6, add reinforcement: "This phase is NOT optional. Every run with a ship or ship-with-gaps verdict MUST reach this point."

  **Verification:**
  - Mandatory checkpoint text appears between Phase 5.5 and Phase 6
  - Phase 6 has reinforcement text that it's not optional
